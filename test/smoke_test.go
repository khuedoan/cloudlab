package e2e

import (
	"errors"
	"net/http"
	"os"
	"testing"
)

func assertStatusCode(t *testing.T, url string, expectedStatusCode int) {
	t.Helper()
	assertStatusCodeWithRedirects(t, url, true, expectedStatusCode)
}

func assertStatusCodeWithRedirects(t *testing.T, url string, followRedirects bool, expectedStatusCodes ...int) {
	t.Helper()

	client := &http.Client{}
	if !followRedirects {
		client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	resp, err := client.Get(url)
	if err != nil {
		if !errors.Is(err, http.ErrUseLastResponse) {
			t.Fatal(err)
		}
	}

	if resp == nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	for _, expectedStatusCode := range expectedStatusCodes {
		if resp.StatusCode == expectedStatusCode {
			return
		}
	}

	t.Fatalf("expected status code %v, but got %d for %s", expectedStatusCodes, resp.StatusCode, url)
}

func TestBlog(t *testing.T) {
	assertStatusCode(t, "https://khuedoan.com", http.StatusOK) // TODO get domain name automatically
}

func TestHomelabDocs(t *testing.T) {
	assertStatusCode(t, "https://homelab.khuedoan.com", http.StatusOK) // TODO get domain name automatically
}

// TODO remove env-specific stuff once we migrate everything to the same setup
func TestStaging(t *testing.T) {
	if os.Getenv("CLOUDLAB_ENV") != "staging" && os.Getenv("SMOKE_STAGING") == "" {
		t.Skip("set CLOUDLAB_ENV=staging or SMOKE_STAGING=1 to run staging smoke tests")
	}

	testCases := []struct {
		name               string
		url                string
		expectedStatusCode int
	}{
		{
			name:               "Vault",
			url:                "https://vault.staging.khuedoan.com",
			expectedStatusCode: http.StatusTemporaryRedirect,
		},
		{
			name:               "Dex",
			url:                "https://dex.staging.khuedoan.com",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Forgejo",
			url:                "https://code.staging.khuedoan.com",
			expectedStatusCode: http.StatusSeeOther,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assertStatusCodeWithRedirects(t, testCase.url, false, testCase.expectedStatusCode)
		})
	}
}
