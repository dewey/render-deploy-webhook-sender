package rendercom

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type RenderService struct {
	c *http.Client
}

type AddHeaderTransport struct {
	T        http.RoundTripper
	apiToken string
}

func NewCustomRoundTripper(apiToken string) *AddHeaderTransport {
	return &AddHeaderTransport{
		T:        http.DefaultTransport,
		apiToken: apiToken,
	}
}

func (adt *AddHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", adt.apiToken))
	req.Header.Add("Accept", "application/json")
	return adt.T.RoundTrip(req)
}

func NewRenderService(c *http.Client, apiToken string) RenderService {
	c.Transport = NewCustomRoundTripper(apiToken)
	return RenderService{c: c}
}

func (rs *RenderService) GetServices() ([]Service, error) {
	resp, err := rs.c.Get("https://api.render.com/v1/services")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d, wanted %d", resp.StatusCode, http.StatusOK)
	}
	var s []Service
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return nil, err
	}
	return s, nil
}

func (rs *RenderService) GetDeploys(serviceId string, status string) ([]Deploy, error) {
	resp, err := rs.c.Get(fmt.Sprintf("https://api.render.com/v1/services/%s/deploys", serviceId))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d, wanted %d", resp.StatusCode, http.StatusOK)
	}
	var d []Deploy
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return nil, err
	}
	// If we don't have a filter, return all
	if status == "" {
		return d, nil
	}
	// Only return filtered deploys
	var filtered []Deploy
	for _, deploy := range d {
		if deploy.Deploy.Status == status {
			filtered = append(filtered, deploy)
		}
	}
	return filtered, nil
}

type Service struct {
	Cursor  string `json:"cursor"`
	Service struct {
		Id             string        `json:"id"`
		AutoDeploy     string        `json:"autoDeploy"`
		Branch         string        `json:"branch"`
		CreatedAt      time.Time     `json:"createdAt"`
		Name           string        `json:"name"`
		NotifyOnFail   string        `json:"notifyOnFail"`
		OwnerId        string        `json:"ownerId"`
		Repo           string        `json:"repo"`
		Slug           string        `json:"slug"`
		Suspended      string        `json:"suspended"`
		Suspenders     []interface{} `json:"suspenders"`
		Type           string        `json:"type"`
		UpdatedAt      time.Time     `json:"updatedAt"`
		ServiceDetails struct {
			BuildCommand               string      `json:"buildCommand"`
			ParentServer               interface{} `json:"parentServer"`
			PublishPath                string      `json:"publishPath"`
			PullRequestPreviewsEnabled string      `json:"pullRequestPreviewsEnabled"`
			Url                        string      `json:"url"`
		} `json:"serviceDetails"`
	} `json:"service"`
}

type Deploy struct {
	Deploy struct {
		Id     string `json:"id"`
		Commit struct {
			Id        string    `json:"id"`
			Message   string    `json:"message"`
			CreatedAt time.Time `json:"createdAt"`
		} `json:"commit"`
		Status     string    `json:"status"`
		CreatedAt  time.Time `json:"createdAt"`
		UpdatedAt  time.Time `json:"updatedAt"`
		FinishedAt time.Time `json:"finishedAt"`
	} `json:"deploy"`
	Cursor string `json:"cursor"`
}
