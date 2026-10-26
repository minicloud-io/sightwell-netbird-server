package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Sightwell fork: --upload-bundle has no default service to fall back to.
func TestCheckUploadFlags(t *testing.T) {
	savedUpload, savedURL := uploadBundleFlag, uploadBundleURLFlag
	t.Cleanup(func() { uploadBundleFlag, uploadBundleURLFlag = savedUpload, savedURL })

	tests := []struct {
		name    string
		upload  bool
		url     string
		wantErr bool
	}{
		{name: "no upload", upload: false, url: ""},
		{name: "upload with url", upload: true, url: "https://upload.example/upload-url"},
		{name: "upload without url", upload: true, url: "", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			uploadBundleFlag, uploadBundleURLFlag = tc.upload, tc.url
			err := checkUploadFlags()
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
