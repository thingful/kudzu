package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	"github.com/thingful/kudzu/pkg/logger"
	"github.com/thingful/kudzu/pkg/postgres"
	goji "goji.io"
	"goji.io/pat"
)

var defaultScope = []string{"timeseries"}

// RegisterAppHandlers registers our new app creation endpoint with the mux
func RegisterAppHandlers(mux *goji.Mux, db *postgres.DB) {
	mux.Handle(pat.Post("/apps/new"), Handler{env: &Env{db: db}, handler: createAppHandler})
}

type appRequest struct {
	App struct {
		Name  string   `json:"Name"`
		Scope []string `json:"Scope"`
	} `json:"App"`
}

type appResponse struct {
	App struct {
		APIKey string `json:"ApiKey"`
	} `json:"App"`
}

func createAppHandler(env *Env, w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	log := logger.FromContext(ctx)

	appReq, err := parseCreateAppRequest(r)
	if err != nil {
		return err
	}

	if appReq.App.Name == "" {
		return &HTTPError{
			Code: http.StatusUnprocessableEntity,
			Err:  errors.New("application name must be supplied and be a non-empty string"),
		}
	}

	if len(appReq.App.Scope) == 0 {
		appReq.App.Scope = defaultScope
	}

	app, err := env.db.CreateApp(ctx, appReq.App.Name, appReq.App.Scope)
	if err != nil {
		return &HTTPError{
			Code: http.StatusInternalServerError,
			Err:  errors.Wrap(err, "failed to save app to the database"),
		}
	}

	log.Log(
		"msg", "created app",
		"name", app.Name,
	)

	b, err := json.Marshal(struct {
		APIKey string `json:"ApiKey"`
	}{
		APIKey: app.Key,
	})

	if err != nil {
		return &HTTPError{
			Code: http.StatusInternalServerError,
			Err:  errors.Wrap(err, "failed to marshal response JSON"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(b)

	return nil
}

func parseCreateAppRequest(r *http.Request) (*appRequest, error) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, &HTTPError{
			Code: http.StatusInternalServerError,
			Err:  errors.Wrap(err, "failed to read incoming request body"),
		}
	}

	var data appRequest
	err = json.Unmarshal(b, &data)
	if err != nil {
		return nil, &HTTPError{
			Code: http.StatusUnprocessableEntity,
			Err:  errors.Wrap(err, "failed to parse incoming request body"),
		}
	}

	return &data, nil
}
