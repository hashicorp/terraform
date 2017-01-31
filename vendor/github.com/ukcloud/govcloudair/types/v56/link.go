package types

// LinkPredicate is a predicate for finding links in a link list
type LinkPredicate func(*Link) bool

func byTypeAndRel(tpe, rel string) LinkPredicate {
	if rel == "" {
		rel = RelDown
	}
	return func(lnk *Link) bool {
		return lnk != nil && lnk.Type == tpe && lnk.Rel == rel
	}
}

func byNameTypeAndRel(nme, tpe, rel string) LinkPredicate {
	tpePred := byTypeAndRel(tpe, rel)
	return func(lnk *Link) bool {
		return tpePred(lnk) && lnk.Name == nme
	}
}

// LinkList represents a list of links
type LinkList []*Link

// Find the first occurrence that matches the predicate
func (l LinkList) Find(predicate LinkPredicate) *Link {
	for _, lnk := range l {
		if predicate(lnk) {
			return lnk
		}
	}
	return nil
}

// ForType finds a link for a given type
func (l LinkList) ForType(tpe, rel string) *Link {
	return l.Find(byTypeAndRel(tpe, rel))
}

// ForName finds a link for a given name and type
func (l LinkList) ForName(name, tpe, rel string) *Link {
	return l.Find(byNameTypeAndRel(name, tpe, rel))
}

const (
	RelDown          = "down"
	RelAdd           = "add"
	RelUp            = "up"
	RelEdit          = "edit"
	RelRemove        = "remove"
	RelCopy          = "copy"
	RelMove          = "move"
	RelAlternate     = "alternate"
	RelTaskCancel    = "task:cancel"
	RelDeploy        = "deploy"
	RelUndeploy      = "undeploy"
	RelDiscardState  = "discardState"
	RelPowerOn       = "power:powerOn"
	RelPowerOff      = "power:powerOff"
	RelPowerReset    = "power:reset"
	RelPowerReboot   = "power:reboot"
	RelPowerSuspend  = "power:suspend"
	RelPowerShutdown = "power:shutdown"

	RelScreenThumbnail        = "screen:thumbnail"
	RelScreenAcquireTicket    = "screen:acquireTicket"
	RelScreenAcquireMksTicket = "screen:acquireMksTicket"

	RelMediaInsertMedia = "media:insertMedia"
	RelMediaEjectMedia  = "media:ejectMedia"

	RelDiskAttach = "disk:attach"
	RelDiskDetach = "disk:detach"

	RelUploadDefault   = "upload:default"
	RelUploadAlternate = "upload:alternate"

	RelDownloadDefault   = "download:default"
	RelDownloadAlternate = "download:alternate"
	RelDownloadIdentity  = "download:identity"

	RelSnapshotCreate          = "snapshot:create"
	RelSnapshotRevertToCurrent = "snapshot:revertToCurrent"
	RelSnapshotRemoveAll       = "snapshot:removeAll"

	RelOVF               = "ovf"
	RelOVA               = "ova"
	RelControlAccess     = "controlAccess"
	RelPublish           = "publish"
	RelPublishExternal   = "publishToExternalOrganizations"
	RelSubscribeExternal = "subscribeToExternalCatalog"
	RelExtension         = "extension"
	RelEnable            = "enable"
	RelDisable           = "disable"
	RelMerge             = "merge"
	RelCatalogItem       = "catalogItem"
	RelRecompose         = "recompose"
	RelRegister          = "register"
	RelUnregister        = "unregister"
	RelRepair            = "repair"
	RelReconnect         = "reconnect"
	RelDisconnect        = "disconnect"
	RelUpgrade           = "upgrade"
	RelAnswer            = "answer"
	RelAddOrgs           = "addOrgs"
	RelRemoveOrgs        = "removeOrgs"
	RelSync              = "sync"

	RelVSphereWebClientURL = "vSphereWebClientUrl"
	RelVimServerDvSwitches = "vimServerDvSwitches"

	RelCollaborationResume    = "resume"
	RelCollaborationAbort     = "abort"
	RelCollaborationFail      = "fail"
	RelEnterMaintenanceMode   = "enterMaintenanceMode"
	RelExitMaintenanceMode    = "exitMaintenanceMode"
	RelTask                   = "task"
	RelTaskOwner              = "task:owner"
	RelPreviousPage           = "previousPage"
	RelNextPage               = "nextPage"
	RelFirstPage              = "firstPage"
	RelLastPage               = "lastPage"
	RelInstallVMWareTools     = "installVmwareTools"
	RelConsolidate            = "consolidate"
	RelEntity                 = "entity"
	RelEntityResolver         = "entityResolver"
	RelRelocate               = "relocate"
	RelBlockingTasks          = "blockingTasks"
	RelUpdateProgress         = "updateProgress"
	RelSyncSyslogSettings     = "syncSyslogSettings"
	RelTakeOwnership          = "takeOwnership"
	RelUnlock                 = "unlock"
	RelShadowVMs              = "shadowVms"
	RelTest                   = "test"
	RelUpdateResourcePools    = "update:resourcePools"
	RelRemoveForce            = "remove:force"
	RelStorageClass           = "storageProfile"
	RelRefreshStorageClasses  = "refreshStorageProfile"
	RelRefreshVirtualCenter   = "refreshVirtualCenter"
	RelCheckCompliance        = "checkCompliance"
	RelForceFullCustomization = "customizeAtNextPowerOn"
	RelReloadFromVC           = "reloadFromVc"
	RelMetricsDayView         = "interval:day"
	RelMetricsWeekView        = "interval:week"
	RelMetricsMonthView       = "interval:month"
	RelMetricsYearView        = "interval:year"
	RelMetricsPreviousRange   = "range:previous"
	RelMetricsNextRange       = "range:next"
	RelMetricsLatestRange     = "range:latest"
	RelRights                 = "rights"
	RelMigratVMs              = "migrateVms"
	RelResourcePoolVMList     = "resourcePoolVmList"
	RelCreateEvent            = "event:create"
	RelCreateTask             = "task:create"
	RelUploadBundle           = "bundle:upload"
	RelCleanupBundles         = "bundles:cleanup"
	RelAuthorizationCheck     = "authorization:check"
	RelCleanupRights          = "rights:cleanup"

	RelEdgeGatewayRedeploy           = "edgeGateway:redeploy"
	RelEdgeGatewayReapplyServices    = "edgeGateway:reapplyServices"
	RelEdgeGatewayConfigureServices  = "edgeGateway:configureServices"
	RelEdgeGatewayConfigureSyslog    = "edgeGateway:configureSyslogServerSettings"
	RelEdgeGatewaySyncSyslogSettings = "edgeGateway:syncSyslogSettings"
	RelEdgeGatewayUpgrade            = "edgeGateway:upgrade"
	RelEdgeGatewayUpgradeNetworking  = "edgeGateway:convertToAdvancedNetworking"
	RelVDCManageFirewall             = "manageFirewall"

	RelCertificateUpdate = "certificate:update"
	RelCertificateReset  = "certificate:reset"
	RelTruststoreUpdate  = "truststore:update"
	RelTruststoreReset   = "truststore:reset"
	RelKeyStoreUpdate    = "keystore:update"
	RelKeystoreReset     = "keystore:reset"
	RelKeytabUpdate      = "keytab:update"
	RelKeytabReset       = "keytab:reset"

	RelServiceLinks             = "down:serviceLinks"
	RelAPIFilters               = "down:apiFilters"
	RelResourceClasses          = "down:resourceClasses"
	RelResourceClassActions     = "down:resourceClassActions"
	RelServices                 = "down:services"
	RelACLRules                 = "down:aclRules"
	RelFileDescriptors          = "down:fileDescriptors"
	RelAPIDefinitions           = "down:apiDefinitions"
	RelServiceResources         = "down:serviceResources"
	RelExtensibility            = "down:extensibility"
	RelAPIServiceQuery          = "down:service"
	RelAPIDefinitionsQuery      = "down:apidefinitions"
	RelAPIFilesQuery            = "down:files"
	RelServiceOfferings         = "down:serviceOfferings"
	RelServiceOfferingInstances = "down:serviceOfferingInstances"
	RelHybrid                   = "down:hybrid"

	RelServiceRefresh      = "service:refresh"
	RelServiceAssociate    = "service:associate"
	RelServiceDisassociate = "service:disassociate"

	RelReconfigureVM = "reconfigureVM"

	RelOrgVDCGateways = "edgeGateways"
	RelOrgVDCNetworks = "orgVdcNetworks"

	RelHybridAcquireControlTicket = "hybrid:acquireControlTicket"
	RelHybridAcquireTicket        = "hybrid:acquireTicket"
	RelHybridRefreshTunnel        = "hybrid:refreshTunnel"

	RelMetrics = "metrics"

	RelFederationRegenerateCertificate = "federation:regenerateFederationCertificate"
	RelTemplateInstantiate             = "instantiate"
)
