package applicationbits

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"os"
	"time"

	"code.cloudfoundry.org/cli/cf/api/resources"
	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	. "code.cloudfoundry.org/cli/cf/i18n"
	"code.cloudfoundry.org/cli/cf/net"
	"code.cloudfoundry.org/gofileutils/fileutils"
)

const (
	DefaultAppUploadBitsTimeout = 15 * time.Minute
)

//go:generate counterfeiter . Repository

type Repository interface {
	GetApplicationFiles(appFilesRequest []resources.AppFileResource) ([]resources.AppFileResource, error)
	UploadBits(appGUID string, zipFile *os.File, presentFiles []resources.AppFileResource) (apiErr error)
}

type CloudControllerApplicationBitsRepository struct {
	config  coreconfig.Reader
	gateway net.Gateway
}

func NewCloudControllerApplicationBitsRepository(config coreconfig.Reader, gateway net.Gateway) (repo CloudControllerApplicationBitsRepository) {
	repo.config = config
	repo.gateway = gateway
	return
}

func (repo CloudControllerApplicationBitsRepository) UploadBits(appGUID string, zipFile *os.File, presentFiles []resources.AppFileResource) (apiErr error) {
	apiURL := fmt.Sprintf("/v2/apps/%s/bits", appGUID)
	fileutils.TempFile("requests", func(requestFile *os.File, err error) {
		if err != nil {
			apiErr = fmt.Errorf("%s: %s", T("Error creating tmp file: {{.Err}}", map[string]interface{}{"Err": err}), err.Error())
			return
		}

		// json.Marshal represents a nil value as "null" instead of an empty slice "[]"
		if presentFiles == nil {
			presentFiles = []resources.AppFileResource{}
		}

		presentFilesJSON, err := json.Marshal(presentFiles)
		if err != nil {
			apiErr = fmt.Errorf("%s: %s", T("Error marshaling JSON"), err.Error())
			return
		}

		boundary, err := repo.writeUploadBody(zipFile, requestFile, presentFilesJSON)
		if err != nil {
			apiErr = fmt.Errorf("%s: %s", T("Error writing to tmp file: {{.Err}}", map[string]interface{}{"Err": err}), err.Error())
			return
		}

		var request *net.Request
		request, apiErr = repo.gateway.NewRequestForFile("PUT", repo.config.APIEndpoint()+apiURL, repo.config.AccessToken(), requestFile)
		if apiErr != nil {
			return
		}

		contentType := fmt.Sprintf("multipart/form-data; boundary=%s", boundary)
		request.HTTPReq.Header.Set("Content-Type", contentType)

		response := &resources.Resource{}
		_, apiErr = repo.gateway.PerformPollingRequestForJSONResponse(repo.config.APIEndpoint(), request, response, DefaultAppUploadBitsTimeout)
		if apiErr != nil {
			return
		}
	})

	return
}

func (repo CloudControllerApplicationBitsRepository) GetApplicationFiles(appFilesToCheck []resources.AppFileResource) ([]resources.AppFileResource, error) {
	integrityFieldsJSON, err := json.Marshal(mapAppFilesToIntegrityFields(appFilesToCheck))
	if err != nil {
		apiErr := fmt.Errorf("%s: %s", T("Failed to create json for resource_match request"), err.Error())
		return nil, apiErr
	}

	responseFieldsColl := []resources.IntegrityFields{}
	apiErr := repo.gateway.UpdateResourceSync(
		repo.config.APIEndpoint(),
		"/v2/resource_match",
		bytes.NewReader(integrityFieldsJSON),
		&responseFieldsColl)

	if apiErr != nil {
		return nil, apiErr
	}

	return intersectAppFilesIntegrityFields(appFilesToCheck, responseFieldsColl), nil
}

func mapAppFilesToIntegrityFields(in []resources.AppFileResource) (out []resources.IntegrityFields) {
	for _, appFile := range in {
		out = append(out, appFile.ToIntegrityFields())
	}
	return out
}

func intersectAppFilesIntegrityFields(
	appFiles []resources.AppFileResource,
	integrityFields []resources.IntegrityFields,
) (out []resources.AppFileResource) {
	inputFiles := appFilesBySha(appFiles)
	for _, responseFields := range integrityFields {
		item, found := inputFiles[responseFields.Sha1]
		if found {
			out = append(out, item)
		}
	}
	return out
}

func appFilesBySha(in []resources.AppFileResource) (out map[string]resources.AppFileResource) {
	out = map[string]resources.AppFileResource{}
	for _, inputFileResource := range in {
		out[inputFileResource.Sha1] = inputFileResource
	}
	return out
}

func (repo CloudControllerApplicationBitsRepository) writeUploadBody(zipFile *os.File, body *os.File, presentResourcesJSON []byte) (boundary string, err error) {
	writer := multipart.NewWriter(body)
	defer writer.Close()

	boundary = writer.Boundary()

	part, err := writer.CreateFormField("resources")
	if err != nil {
		return
	}

	_, err = io.Copy(part, bytes.NewBuffer(presentResourcesJSON))
	if err != nil {
		return
	}

	if zipFile != nil {
		zipStats, zipErr := zipFile.Stat()
		if zipErr != nil {
			return
		}

		if zipStats.Size() == 0 {
			return
		}

		part, zipErr = createZipPartWriter(zipStats, writer)
		if zipErr != nil {
			return
		}

		_, zipErr = io.Copy(part, zipFile)
		if zipErr != nil {
			return
		}
	}

	return
}

func createZipPartWriter(zipStats os.FileInfo, writer *multipart.Writer) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="application"; filename="application.zip"`)
	h.Set("Content-Type", "application/zip")
	h.Set("Content-Length", fmt.Sprintf("%d", zipStats.Size()))
	h.Set("Content-Transfer-Encoding", "binary")
	return writer.CreatePart(h)
}
