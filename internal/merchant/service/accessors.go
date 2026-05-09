package merchantsvc

// StoreBaseURL returns the configured store base URL.
// Used by the handler to pass into feedxml.BuildRSSFeed.
func (s *MerchantFeedService) StoreBaseURL() string {
	return s.cfg.StoreBaseURL
}

// StoreTitle returns the store display name for the feed <channel><title>.
func (s *MerchantFeedService) StoreTitle() string {
	return "Zentora Shop"
}