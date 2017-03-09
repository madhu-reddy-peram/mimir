package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/weaveworks/cortex/configs"
	"github.com/weaveworks/cortex/configs/api"
)

const (
	endpoint        = "/api/configs/org/cortex"
	privateEndpoint = "/private/api/configs/org/cortex"
)

// The root page returns 200 OK.
func Test_Root_OK(t *testing.T) {
	setup(t)
	defer cleanup(t)

	w := request(t, "GET", "/", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

// postConfig posts an organisation config.
func postConfig(t *testing.T, orgID configs.OrgID, config configs.Config) configs.ConfigView {
	w := requestAsOrg(t, orgID, "POST", endpoint, jsonObject(config).Reader(t))
	require.Equal(t, http.StatusNoContent, w.Code)
	return getConfig(t, orgID)
}

// getConfig gets an organisation config.
func getConfig(t *testing.T, orgID configs.OrgID) configs.ConfigView {
	w := requestAsOrg(t, orgID, "GET", endpoint, nil)
	return parseConfigView(t, w.Body.Bytes())
}

// configs returns 401 to requests without authentication.
func Test_GetConfig_Anonymous(t *testing.T) {
	setup(t)
	defer cleanup(t)

	w := request(t, "GET", endpoint, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// configs returns 404 if there's no such subsystem.
func Test_GetConfig_NotFound(t *testing.T) {
	setup(t)
	defer cleanup(t)

	orgID := makeOrgID()
	w := requestAsOrg(t, orgID, "GET", endpoint, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// configs returns 401 to requests without authentication.
func Test_PostConfig_Anonymous(t *testing.T) {
	setup(t)
	defer cleanup(t)

	w := request(t, "POST", endpoint, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// Posting to a configuration sets it so that you can get it again.
func Test_PostConfig_CreatesConfig(t *testing.T) {
	setup(t)
	defer cleanup(t)

	orgID := makeOrgID()
	config := makeConfig()
	content := jsonObject(config)
	{
		w := requestAsOrg(t, orgID, "POST", endpoint, content.Reader(t))
		assert.Equal(t, http.StatusNoContent, w.Code)
	}
	{
		w := requestAsOrg(t, orgID, "GET", endpoint, nil)
		assert.Equal(t, config, parseConfigView(t, w.Body.Bytes()).Config)
	}
}

// Posting to a configuration sets it so that you can get it again.
func Test_PostConfig_UpdatesConfig(t *testing.T) {
	setup(t)
	defer cleanup(t)

	orgID := makeOrgID()
	view1 := postConfig(t, orgID, makeConfig())
	config2 := makeConfig()
	view2 := postConfig(t, orgID, config2)
	assert.True(t, view2.ID > view1.ID, "%v > %v", view2.ID, view1.ID)
	assert.Equal(t, config2, view2.Config)
}

// Different users can have different configurations.
func Test_PostConfig_MultipleOrgs(t *testing.T) {
	setup(t)
	defer cleanup(t)

	orgID1 := makeOrgID()
	orgID2 := makeOrgID()
	config1 := postConfig(t, orgID1, makeConfig())
	config2 := postConfig(t, orgID2, makeConfig())

	foundConfig1 := getConfig(t, orgID1)
	assert.Equal(t, config1, foundConfig1)
	foundConfig2 := getConfig(t, orgID2)
	assert.Equal(t, config2, foundConfig2)
	assert.True(t, config2.ID > config1.ID, "%v > %v", config2.ID, config1.ID)
}

// GetAllConfigs returns an empty list of configs if there aren't any.
func Test_GetAllConfigs_Empty(t *testing.T) {
	setup(t)
	defer cleanup(t)

	w := request(t, "GET", privateEndpoint, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var found api.ConfigsView
	err := json.Unmarshal(w.Body.Bytes(), &found)
	assert.NoError(t, err, "Could not unmarshal JSON")
	assert.Equal(t, api.ConfigsView{Configs: map[configs.OrgID]configs.ConfigView{}}, found)
}

// GetAllConfigs returns all created configs.
func Test_GetAllConfigs(t *testing.T) {
	setup(t)
	defer cleanup(t)

	orgID := makeOrgID()
	config := makeConfig()
	view := postConfig(t, orgID, config)
	w := request(t, "GET", privateEndpoint, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var found api.ConfigsView
	err := json.Unmarshal(w.Body.Bytes(), &found)
	assert.NoError(t, err, "Could not unmarshal JSON")
	assert.Equal(t, api.ConfigsView{Configs: map[configs.OrgID]configs.ConfigView{
		orgID: view,
	}}, found)
}

// GetAllConfigs returns the *newest* versions of all created configs.
func Test_GetAllConfigs_Newest(t *testing.T) {
	setup(t)
	defer cleanup(t)

	orgID := makeOrgID()
	postConfig(t, orgID, makeConfig())
	postConfig(t, orgID, makeConfig())
	lastCreated := postConfig(t, orgID, makeConfig())

	w := request(t, "GET", privateEndpoint, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var found api.ConfigsView
	err := json.Unmarshal(w.Body.Bytes(), &found)
	assert.NoError(t, err, "Could not unmarshal JSON")
	assert.Equal(t, api.ConfigsView{Configs: map[configs.OrgID]configs.ConfigView{
		orgID: lastCreated,
	}}, found)
}

func Test_GetConfigs_IncludesNewerConfigsAndExcludesOlder(t *testing.T) {
	setup(t)
	defer cleanup(t)

	postConfig(t, makeOrgID(), makeConfig())
	config2 := postConfig(t, makeOrgID(), makeConfig())
	orgID3 := makeOrgID()
	config3 := postConfig(t, orgID3, makeConfig())

	w := request(t, "GET", fmt.Sprintf("%s?since=%d", privateEndpoint, config2.ID), nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var found api.ConfigsView
	err := json.Unmarshal(w.Body.Bytes(), &found)
	assert.NoError(t, err, "Could not unmarshal JSON")
	assert.Equal(t, api.ConfigsView{Configs: map[configs.OrgID]configs.ConfigView{
		orgID3: config3,
	}}, found)
}
