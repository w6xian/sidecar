package services

import (
	"bytes"
	"io"
	"net/http"
)

func (svr *Service) PostData() ([]byte, error) {
	r := bytes.NewReader(svr.Payload)
	req, err := http.NewRequest(http.MethodPost, svr.Url, r)
	if err != nil {
		return nil, err
	}
	for k, v := range svr.Header {
		req.Header.Add(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close() // 这步是必要的，防止以后的内存泄漏，切记
	resData, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return resData, nil
}
