package uploaders

import (
	"fmt"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-xcode/exportoptions"
	"github.com/bitrise-io/go-xcode/ipa"
	"github.com/bitrise-io/go-xcode/plistutil"
	"github.com/bitrise-io/go-xcode/profileutil"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/deployment"
)

// DeployIPA ...
func DeployIPA(item deployment.DeployableItem, buildURL, token, notifyUserGroups, notifyEmails string, isEnablePublicPage bool) (ArtifactURLs, error) {
	pth := item.Path
	infoPlistPth, err := ipa.UnwrapEmbeddedInfoPlist(pth)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to unwrap Info.plist from ipa, error: %s", err)
	}

	infoPlistData, err := plistutil.NewPlistDataFromFile(infoPlistPth)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to parse Info.plist, error: %s", err)
	}

	appTitle, _ := infoPlistData.GetString("CFBundleName")
	appDisplayNameTitle, _ := infoPlistData.GetString("CFBundleDisplayName")
	bundleID, _ := infoPlistData.GetString("CFBundleIdentifier")
	version, _ := infoPlistData.GetString("CFBundleShortVersionString")
	buildNumber, _ := infoPlistData.GetString("CFBundleVersion")
	minOSVersion, _ := infoPlistData.GetString("MinimumOSVersion")
	deviceFamilyList, _ := infoPlistData.GetUInt64Array("UIDeviceFamily")

	bestAppTitle = appTitle
	if appDisplayNameTitle != nil {
		bestAppTitle = appDisplayNameTitle
	}

	appInfo := map[string]interface{}{
		"app_title":          bestAppTitle,
		"bundle_id":          bundleID,
		"version":            version,
		"build_number":       buildNumber,
		"min_OS_version":     minOSVersion,
		"device_family_list": deviceFamilyList,
	}

	log.Printf("ipa infos: %v", appInfo)

	// ---

	provisioningProfilePth, err := ipa.UnwrapEmbeddedMobileProvision(pth)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to unwrap embedded.mobilprovision from ipa, error: %s", err)
	}

	provisioningProfileInfo, err := profileutil.NewProvisioningProfileInfoFromFile(provisioningProfilePth)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to parse embedded.mobilprovision, error: %s", err)
	}

	teamName := provisioningProfileInfo.TeamName
	creationDate := provisioningProfileInfo.CreationDate
	provisionsAlldevices := provisioningProfileInfo.ProvisionsAllDevices
	expirationDate := provisioningProfileInfo.ExpirationDate
	deviceUDIDList := provisioningProfileInfo.ProvisionedDevices
	profileName := provisioningProfileInfo.Name
	exportMethod := provisioningProfileInfo.ExportType

	if exportMethod == exportoptions.MethodAppStore {
		log.Warnf("is_enable_public_page is set, but public download isn't allowed for app-store distributions")
		log.Warnf("setting is_enable_public_page to false ...")
		isEnablePublicPage = false
	}

	provisioningInfo := map[string]interface{}{
		"creation_date":          creationDate,
		"expire_date":            expirationDate,
		"device_UDID_list":       deviceUDIDList,
		"team_name":              teamName,
		"profile_name":           profileName,
		"provisions_all_devices": provisionsAlldevices,
		"ipa_export_method":      exportMethod,
	}

	// ---

	fileSize, err := fileSizeInBytes(pth)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to get ipa size, error: %s", err)
	}

	ipaInfoMap := map[string]interface{}{
		"file_size_bytes":   fmt.Sprintf("%f", fileSize),
		"app_info":          appInfo,
		"provisioning_info": provisioningInfo,
	}

	// ---

	const IPAContentType = "application/octet-stream ipa"
	uploadURL, artifactID, err := createArtifact(buildURL, token, pth, "ios-ipa", IPAContentType)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to create ipa artifact: %s %w", pth, err)
	}

	if err := UploadArtifact(uploadURL, pth, IPAContentType); err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to upload ipa artifact, error: %s", err)
	}

	buildArtifactMeta := AppDeploymentMetaData{
		ArtifactInfo:       ipaInfoMap,
		NotifyUserGroups:   notifyUserGroups,
		NotifyEmails:       notifyEmails,
		IsEnablePublicPage: isEnablePublicPage,
	}

	artifactURLs, err := finishArtifact(buildURL, token, artifactID, &buildArtifactMeta, item.IntermediateFileMeta)
	if err != nil {
		return ArtifactURLs{}, fmt.Errorf("failed to finish ipa artifact, error: %s", err)
	}

	return artifactURLs, nil
}
