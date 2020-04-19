package model

// AuthBackend is the backend that has the auth system.
type AuthBackend struct {
	ID string

	Dex *AuthBackendDex
}

// AuthBackendDex is the configuraiton of dex AuthBackend.
type AuthBackendDex struct {
	APIURL    string
	PublicURL string
}

// App is a representation of an app that wants to be secured.
type App struct {
	ID            string
	AuthBackendID string
	Host          string
	UpstreamURL   string
}
