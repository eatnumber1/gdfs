package main

import (
	"flag"
	"log"
	"fmt"
	"net/http"

	"github.com/eatnumber1/gdfs/fs"

	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"

	"code.google.com/p/google-api-go-client/drive/v2"
	"code.google.com/p/goauth2/oauth"
)

var (
	cachefile = flag.String("cache", "cache.json", "Token cache file")
)

// Settings for authorization.
var config = &oauth.Config{
	ClientId: "12763834838-cbckm8j2p4gesmdrok8censnqn0mahcu.apps.googleusercontent.com",
	ClientSecret: "BNnTKWcgj9FyANzbuRrjcolT",
	Scope: "https://www.googleapis.com/auth/drive",
	RedirectURL: "urn:ietf:wg:oauth:2.0:oob",
	AuthURL: "https://accounts.google.com/o/oauth2/auth",
	TokenURL: "https://accounts.google.com/o/oauth2/token",
	TokenCache: oauth.CacheFile(*cachefile),
}

func main() {
	flag.Parse()

	transport := &oauth.Transport{
		Config: config,
		Transport: http.DefaultTransport,
	}

	// Try to pull the token from the cache; if this fails, we need to get one.
	token, err := config.TokenCache.Token()
	if err != nil {

		// Generate a URL to visit for authorization.
		authUrl := config.AuthCodeURL("state")
		log.Printf("Go to the following link in your browser: %v\n", authUrl)

		// Read the code, and exchange it for a token.
		log.Printf("Enter verification code: ")
		var code string
		fmt.Scanln(&code)

		// Exchange the authorization code for an access token.
		// ("Here's the code you gave the user, now give me a token!")
		token, err = transport.Exchange(code)
		if err != nil {
			log.Fatal("Exchange:", err)
		}
		// (The Exchange method will automatically cache the token.)
		fmt.Printf("Token is cached in %v\n", config.TokenCache)
	}

	transport.Token = token

	// Create a new authorized Drive client.
	svc, err := drive.New(transport.Client())
	if err != nil {
		log.Fatalf("An error occurred creating Drive client: %v\n", err)
	}

	gdfs, err := gdfs.NewGdfs(pathfs.NewDefaultFileSystem(), svc)
	if err != nil {
		log.Fatalf("Cannot construct Gdfs: %v\n", err)
	}

	nfs := pathfs.NewPathNodeFs(gdfs, nil)
	server, _, err := nodefs.MountRoot(flag.Arg(0), nfs.Root(), nil)
	if err != nil {
		log.Fatalf("Mount fail: %v\n", err)
	}
	server.Serve()
}
