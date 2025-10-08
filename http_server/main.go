package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const CLRF = "\r\n"

type Headers map[string]string

type Request struct {
	Method      string
	Target      string
	HttpVersion string
	Headers     Headers
	Body        []byte
}

func ParseRequest(rawRequest []byte) (*Request, error) {
	req := &Request{
		Headers: make(Headers),
	}

	reader := bufio.NewReader(bytes.NewReader(rawRequest))

	// ========================================
	// start: Parse Request Line
	// ========================================
	reqLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("Failed to read request line: %w", err)
	}
	reqLine = strings.TrimSpace(reqLine) // remove clrf

	parts := strings.Split(reqLine, " ")
	if len(parts) != 3 {
		return nil, fmt.Errorf("Invalid request line format: %s", reqLine)
	}
	req.Method = parts[0]
	req.Target = parts[1]
	req.HttpVersion = parts[2]
	// ========================================
	// end: Parse Request Line
	// ========================================

	// ========================================
	// start: Parse Headers
	// ========================================
	for {
		headerLine, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF { // EOF is expected if there's no body
				break
			}
			return nil, fmt.Errorf("Failed to read header line: %w", err)
		}
		headerLine = strings.TrimSpace(headerLine)

		// empty line signifies end of headers
		if headerLine == "" {
			break
		}

		headerParts := strings.SplitN(headerLine, ":", 2)
		if len(headerParts) != 2 {
			fmt.Printf("Malformed header line found: %s\n", headerLine)
			continue
		}
		key := strings.TrimSpace(headerParts[0])
		value := strings.TrimSpace(headerParts[1])
		req.Headers[key] = value
	}
	// ========================================
	// end: Parse Headers
	// ========================================

	// ========================================
	// start: Parse Body
	// ========================================
	remainingBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("Failed to read request body: %w", err)
	}
	req.Body = remainingBytes
	// ========================================
	// end: Parse Body
	// ========================================

	return req, nil
}

type ResponseParam struct {
	Status  string
	Headers Headers
	Body    []byte
}

func CreateResponse(params ResponseParam) []byte {
	if params.Status == "" {
		fmt.Println("Warning status line is empty.")
	}

	if _, ok := params.Headers["Content-Length"]; !ok {
		params.Headers["Content-Length"] = strconv.Itoa(len(params.Body))
	}

	var sb strings.Builder
	sb.WriteString(params.Status)
	sb.WriteString(CLRF)
	for key, value := range params.Headers {
		sb.WriteString(fmt.Sprintf("%s: %s", key, value))
		sb.WriteString(CLRF)
	}
	sb.WriteString(CLRF)
	sb.Write(params.Body)

	return []byte(sb.String())
}

func MatchPath(pattern string, target string) (bool, []string) {
	re := regexp.MustCompile(pattern)

	if re.MatchString(target) {
		matches := re.FindStringSubmatch(target)

		if len(matches) <= 0 {
			fmt.Printf("Target: '%s' does not match the pattern.\n", target)
		}

		return true, matches
	}

	return false, nil
}

func GzipBytes(in []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	if err != nil {
		return nil, err
	}
	if _, err := zw.Write(in); err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func handleEncoding(req *Request, resp *ResponseParam) error {
	acceptEncodings := strings.Split(req.Headers["Accept-Encoding"], ", ")
	hasGzip := false
	for _, en := range acceptEncodings {
		if en == "gzip" {
			hasGzip = true
		}
	}
	if hasGzip {
		resp.Headers["Content-Encoding"] = "gzip"

		if len(resp.Body) > 0 {
			compressed, err := GzipBytes(resp.Body)
			if err != nil {
				return err
			}
			resp.Headers["Content-Length"] = strconv.Itoa(len(compressed))
			resp.Body = compressed
		}
	} else {
		delete(resp.Headers, "Content-Encoding")
	}

	return nil
}

func handleRequest(c net.Conn, dir *string) {
	defer c.Close()

	reqBuf := make([]byte, 65536)
	n, err := c.Read(reqBuf)
	if err != nil {
		fmt.Println("Error accepting connection: ", err)
	}

	req, err := ParseRequest(reqBuf[:n])
	if err != nil {
		fmt.Println("Error parsing request buffer: ", err)
	}

	respParam := &ResponseParam{
		Status:  "HTTP/1.1 404 Not Found",
		Headers: make(Headers),
		Body:    []byte("Not Found"),
	}

	isEchoPath, matchedEcho := MatchPath(`^\/echo\/(.*)$`, req.Target)
	isFilePath, matchedFile := MatchPath(`^\/files\/(.*)$`, req.Target)

	switch {
	case req.Target == "/":
		respParam.Status = "HTTP/1.1 200 OK"
		respParam.Headers["Content-Type"] = "text/plain"
		respParam.Body = []byte("")

	case req.Target == "/user-agent":
		userAgent := req.Headers["User-Agent"]
		respParam.Status = "HTTP/1.1 200 OK"
		respParam.Headers["Content-Type"] = "text/plain"
		respParam.Body = []byte(userAgent)

	case isEchoPath && len(matchedEcho) > 1:
		matched := matchedEcho[1]
		respParam.Status = "HTTP/1.1 200 OK"
		respParam.Headers["Content-Type"] = "text/plain"
		respParam.Body = []byte(matched)

	case isFilePath && len(matchedFile) > 1:
		filename := matchedFile[1]
		path := *dir + filename

		switch req.Method {
		case "GET":
			fb, err := os.ReadFile(path)
			if err != nil {
				respParam.Status = "HTTP/1.1 404 Not Found"
				respParam.Headers["Content-Type"] = "text/plain"
				respParam.Body = []byte("Not Found")
			} else {
				respParam.Status = "HTTP/1.1 200 OK"
				respParam.Headers["Content-Type"] = "application/octet-stream"
				respParam.Body = fb
			}
		case "POST":
			err := os.WriteFile(path, req.Body, 0644)
			if err != nil {
				respParam.Status = "HTTP/1.1 500 Server Error"
			} else {
				respParam.Status = "HTTP/1.1 201 Created"
				respParam.Headers["Content-Type"] = "application/octet-stream"
				respParam.Body = nil
			}
		}
	}

	handleEncoding(req, respParam)

	resp := CreateResponse(*respParam)
	c.Write(resp)

	handleRequest(c, dir)
}

func main() {
	dir := flag.String("directory", "", "Directory path to serve files from.")
	flag.Parse()

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	defer l.Close()

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handleRequest(c, dir)
	}
}
