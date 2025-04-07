package upload

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
)

// UploadPath := "http://localhost:3000/file/upload"
type media3Repo struct {
	Media3Host string
	Client     *http.Client
	UploadPath string // "/file/upload"
}

type Media3Repository interface {
	UploadImage(file, query string) (string, error)
}

func NewMedia3Repository(Media3Host string) Media3Repository {
	return &media3Repo{
		Media3Host: Media3Host,
		Client:     http.DefaultClient,
	}
}

func (m *media3Repo) fileUploadRequest(file []byte, query string) (path string, err error) {
	url := fmt.Sprintf("%v%v", m.Media3Host, m.UploadPath)
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile(query, query)
	if err != nil {
		return
	}
	io.Copy(fw, bytes.NewReader(file))
	w.Close()

	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := m.Client.Do(req)
	if err != nil {
		return
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("bad status: %s", resp.Status)
		return
	}

	body := &bytes.Buffer{}
	_, err = body.ReadFrom(resp.Body)
	if err != nil {
		return "", err
	}
	resp.Body.Close()

	type ResponseMedia struct {
		URL string `json:"url"`
	}

	var media ResponseMedia

	err = json.Unmarshal(body.Bytes(), &media)
	if err != nil {
		return
	}
	path = media.URL
	return
}

func (m *media3Repo) UploadImage(file, query string) (string, error) {
	res, err := http.Get(file)
	if err != nil {
		log.Fatalf("http.Get -> %v", err)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("io.ReadAll -> %v", err)
	}
	res.Body.Close()

	shareLink, err := m.fileUploadRequest(data, query)
	if err != nil {
		return "", nil
	}
	return shareLink, nil
}
