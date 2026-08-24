package platform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"account-switcher/internal/appclient"
	"account-switcher/internal/appconfig"
)

// TranslationLinksDTO tells the settings page which translation features this
// build can actually offer, so it can hide the ones pointing nowhere rather
// than showing a link that opens a blank page.
type TranslationLinksDTO struct {
	// ProjectURL is where "help translate" goes. Empty hides the link.
	ProjectURL string `json:"projectUrl"`
	// CreditsAvailable reports whether translator credits can be fetched.
	CreditsAvailable bool `json:"creditsAvailable"`
}

// GetTranslationLinks reports the translation features available in this build.
func (*PlatformService) GetTranslationLinks() TranslationLinksDTO {
	return TranslationLinksDTO{
		ProjectURL:       appconfig.CrowdinProjectURL,
		CreditsAvailable: appconfig.Configured(appconfig.CrowdinTranslatorsURL),
	}
}

// ErrCreditsNotConfigured means this build has no translator credits service.
var ErrCreditsNotConfigured = errors.New("translator credits are not configured for this build")

// CrowdinProofReader is a project member with proofreader languages.
type CrowdinProofReader struct {
	Name      string `json:"name"`
	Languages string `json:"languages"`
}

// CrowdinTranslatorsList is returned to the SPA for the translators modal.
type CrowdinTranslatorsList struct {
	ProofReaders []CrowdinProofReader `json:"proofReaders"`
	Translators  []string             `json:"translators"`
}

type crowdinAPIResponse struct {
	ProofReaders map[string]string `json:"ProofReaders"`
	Translators  []string          `json:"Translators"`
}

// GetCrowdinTranslators fetches the translator credits.
//
// The credits come from a service the project has to host; a build with none
// configured reports that rather than failing, so the settings page can hide
// the button instead of offering one that always errors.
func (*PlatformService) GetCrowdinTranslators() (CrowdinTranslatorsList, error) {
	if !appconfig.Configured(appconfig.CrowdinTranslatorsURL) {
		return CrowdinTranslatorsList{}, ErrCreditsNotConfigured
	}
	if appclient.IsOfflineMode() {
		return CrowdinTranslatorsList{}, errors.New("offline mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, appconfig.CrowdinTranslatorsURL, nil)
	if err != nil {
		return CrowdinTranslatorsList{}, err
	}

	resp, err := appclient.Shared.Do(req)
	if err != nil {
		return CrowdinTranslatorsList{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return CrowdinTranslatorsList{}, fmt.Errorf("crowdin api: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return CrowdinTranslatorsList{}, err
	}

	var raw crowdinAPIResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return CrowdinTranslatorsList{}, err
	}

	proof := make([]CrowdinProofReader, 0, len(raw.ProofReaders))
	for name, langs := range raw.ProofReaders {
		proof = append(proof, CrowdinProofReader{Name: name, Languages: langs})
	}
	sort.Slice(proof, func(i, j int) bool {
		return proof[i].Name < proof[j].Name
	})

	translators := append([]string(nil), raw.Translators...)
	sort.Strings(translators)

	return CrowdinTranslatorsList{
		ProofReaders: proof,
		Translators:  translators,
	}, nil
}
