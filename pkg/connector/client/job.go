package client

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/conductorone/baton-sdk/pkg/uhttp"
)

func (c *Client) ProcessUserJob(ctx context.Context, job *Job) ([]User, error) {
	if job.Status != "completed" {
		return nil, errors.New("job not completed")
	}

	if job.Location == "" {
		return nil, errors.New("job location is empty")
	}

	urlReq, err := url.Parse(job.Location)
	if err != nil {
		return nil, fmt.Errorf("failed to parse job location: %w", err)
	}

	req, err := c.wrapper.NewRequest(ctx, http.MethodGet, urlReq, []uhttp.RequestOption{}...)
	if err != nil {
		return nil, err
	}

	response, err := c.wrapper.Do(req)
	if response != nil {
		defer response.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch job results: status code %d", response.StatusCode)
	}

	return process(response.Body)
}

func process(reader io.Reader) ([]User, error) {
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()

	allData, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil, err
	}

	index := 0
	lastIndex := 0

	var users []User

	for len(allData) > index {
		if allData[index] != '\n' {
			index++
			continue
		}

		jsonData := allData[lastIndex:index]
		lastIndex = index

		var user User
		err = json.Unmarshal(jsonData, &user)
		if err != nil {
			return nil, err
		}

		users = append(users, user)
		index++
	}

	return users, nil
}
