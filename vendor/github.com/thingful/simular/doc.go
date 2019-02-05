/*
Package simular provides tools for mocking HTTP responses.

Simple Example:
	func TestFetchArticles(t *testing.T) {
		simular.Activate()
		defer simular.DeactivateAndReset()

		simular.RegisterStubRequests(
			simular.NewStubRequest(
				"GET",
				"https://api.mybiz.com/articles.json",
				simular.NewStringResponder(200, `[{"id": 1, "name": "My Great Article"}]`),
			),
		)

		// do stuff that makes a request to articles.json

		// verify that all stubs were called
		if err := simular.AllStubsCalled(); err != nil {
				t.Errorf("Not all stubs were called: %s", err)
		}
	}

Advanced Example:
	func TestFetchArticles(t *testing.T) {
		simular.Activate(
			WithAllowedHosts("localhost"),
		)
		defer simular.DeactivateAndReset()

		// our database of articles
		articles := make([]map[string]interface{}, 0)

		// mock to list out the articles
		simular.RegisterStubRequests(
			simular.NewStubRequest(
				"GET",
				"https://api.mybiz.com/articles.json",
				func(req *http.Request) (*http.Response, error) {
					resp, err := simular.NewJsonResponse(200, articles)
					if err != nil {
						return simular.NewStringResponse(500, ""), nil
					}
					return resp
				},
				simular.WithHeader(
					&http.Header{
						"Api-Key": []string{"1234abcd"},
					},
				),
			),
		)

		// mock to add a new article
		simular.RegisterStubRequests(
			simular.NewStubRequest(
				"POST",
				"https://api.mybiz.com/articles.json",
				func(req *http.Request) (*http.Response, error) {
					article := make(map[string]interface{})
					if err := json.NewDecoder(req.Body).Decode(&article); err != nil {
						return simular.NewStringResponse(400, ""), nil
					}

					articles = append(articles, article)

					resp, err := simular.NewJsonResponse(200, article)
					if err != nil {
						return simular.NewStringResponse(500, ""), nil
					}
					return resp, nil
				},
				simular.WithHeader(
					&http.Header{
						"Api-Key": []string{"1234abcd"},
					},
				),
				simular.WithBody(
					bytes.NewBufferString(`{"title":"article"}`),
				),
			),
		)

		// do stuff that adds and checks articles

		// verify that all stubs were called
		if err := simular.AllStubsCalled(); err != nil {
				t.Errorf("Not all stubs were called: %s", err)
		}
	}

*/
package simular
