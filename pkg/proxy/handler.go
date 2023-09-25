package proxy

// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"istio.io/pkg/log"

	"github.com/DataDog/datadog-go/v5/statsd"

	"github.com/robertcopezd/docker-credential-magic-proxy/internal/config"
	"github.com/robertcopezd/docker-credential-magic-proxy/pkg/common"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
)

var regex = regexp.MustCompile(`/v2/forwardto/(?P<Registry>[^/]+)/(?P<Repository>.+)/(?P<Resource>(manifests|blobs|tags))/(?P<Identifier>.+)$`)

type proxy struct {
	config *config.Data
	dd     *statsd.Client
}

func NewHandler(c *config.Data, dd *statsd.Client) http.Handler {
	return &proxy{config: c, dd: dd}
}

// returns upstream repository and path.
func parsePath(path string, allowHTTP bool) (name.Repository, string, error) {
	matches := regex.FindStringSubmatch(path)
	if matches == nil {
		return name.Repository{}, "", fmt.Errorf("failed to parse the path: %s", path)
	}
	regStr := matches[regex.SubexpIndex("Registry")]
	repoStr := matches[regex.SubexpIndex("Repository")]
	resStr := matches[regex.SubexpIndex("Resource")]
	idenStr := matches[regex.SubexpIndex("Identifier")]
	options := []name.Option{}
	if allowHTTP {
		options = append(options, name.Insecure)
	}
	repo, err := name.NewRepository(fmt.Sprintf("%s/%s", regStr, repoStr), options...)
	if err != nil {
		return name.Repository{}, "", err
	}
	pathForUpstream := fmt.Sprintf("/v2/%s/%s/%s", repoStr, resStr, idenStr)
	return repo, pathForUpstream, nil
}

func (p *proxy) getClient(ctx context.Context, repository name.Repository) (*http.Client, error) {
	// Use just a default key chain.
	// We can use external tools via credHelper config in .docker/config.json
	auth, err := authn.DefaultKeychain.Resolve(repository)
	if err != nil {
		return nil, err
	}

	tr, err := transport.NewWithContext(
		ctx,
		repository.Registry,
		auth,
		remote.DefaultTransport,
		[]string{repository.Scope(transport.PullScope)})
	if err != nil {
		return nil, err
	}

	nc := &http.Client{Transport: tr}
	return nc, nil
}

func (p *proxy) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	//statsd, err := statsd.New(fmt.Sprintf("%s:%d", c.StatsDHost, c.StatsDPort))
	//if err != nil {
	//	return nil, err
	//}

	path := req.URL.Path

	if path == "/v2" || path == "/v2/" {
		// For the fast progess, do this in proxy.
		wr.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
		return
	}

	if req.Header.Get("X-DOCKER-CREDENTIAL-MAGIC-STATUS") != "" {
		wr.WriteHeader(http.StatusLoopDetected)
		return
	}

	repository, upstreamPath, err := parsePath(path, p.config.AllowHTTP)
	if err == nil {
		// If we got http[s]://{server}/forwardto/{host}/{...} , rewrite it http[s]://{host}/{...}
		host := repository.Registry.String()
		// Rewrite the URL
		req.Host = host
		req.URL.Host = host
		// Scheme is automatically determined by `schemeTransport` of go-containerregistry
		req.URL.Path = upstreamPath
	} else {
		// Otherwise, just proxing to "Host".
		repository, err = name.NewRepository(req.Host)
		if err != nil {
			http.Error(wr, err.Error(), http.StatusBadGateway)
			log.Errorf("failed to create client: %v", err)
		}
		req.URL.Host = req.Host
	}

	req.RequestURI = ""
	req.Header.Add("X-DOCKER-CREDENTIAL-MAGIC-STATUS", "done")

	client, err := p.getClient(req.Context(), repository)
	if err != nil {
		http.Error(wr, err.Error(), http.StatusBadGateway)
		log.Errorf("failed to create client: %v", err)
		p.dd.Incr("failure", []string{"error:create_client"}, 1)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		http.Error(wr, "Server Error", http.StatusBadGateway)
		log.Errorf("server error: %v", err)
		p.dd.Incr("failure", []string{"error:server_error"}, 1)
		return
	}
	defer resp.Body.Close()

	log.Infof("%v %s %s %v", resp.Status, req.Host, req.URL.String(), req.RemoteAddr)
	common.CopyHeader(wr.Header(), resp.Header)
	wr.WriteHeader(resp.StatusCode)
	io.Copy(wr, resp.Body)

	if resp.StatusCode == 200 {
		p.dd.Incr("success", nil, 1)
	} else {
		p.dd.Incr("failure", []string{"error:not200"}, 1)
	}
}
