package main

import (
	"flag"
	"log"
	"fmt"
	"net/http"
	"os"
	"encoding/json"
	"io/ioutil"

	//"github.com/eatnumber1/gdfs/fs"
	"github.com/eatnumber1/gdfs/drive"

	"bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"

	gdrive "code.google.com/p/google-api-go-client/drive/v2"
	"code.google.com/p/goauth2/oauth"
)

var (
	cachefile = flag.String("cache", "cache.json", "Token cache file")
)

// Settings for authorization.
var config = &oauth.Config{
	ClientId: "",
	ClientSecret: "",
	Scope: "https://www.googleapis.com/auth/drive",
	RedirectURL: "urn:ietf:wg:oauth:2.0:oob",
	AuthURL: "https://accounts.google.com/o/oauth2/auth",
	TokenURL: "https://accounts.google.com/o/oauth2/token",
	TokenCache: oauth.CacheFile(*cachefile),
}

type Credentials struct {
	Id string
	Secret string
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s [options] MOUNTPOINT\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	mountpoint := flag.Arg(0)

	credsBytes, err := ioutil.ReadFile("creds.json")
	if err != nil {
		log.Fatalf("%v", err)
	}

	var creds Credentials
	err = json.Unmarshal(credsBytes, &creds)
	if err != nil {
		log.Fatalf("%v", err)
	}

	config.ClientSecret = creds.Secret
	config.ClientId = creds.Id

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
	svc, err := gdrive.New(transport.Client())
	if err != nil {
		log.Fatalf("An error occurred creating Drive client: %v\n", err)
	}

	//gdfs, err := gdfs.NewDriveFileSystem(svc, transport.Client())
	gdfs := drive.NewDriveRef(svc, transport.Client())

	c, err := fuse.Mount(mountpoint)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	err = fusefs.Serve(c, gdfs)
	if err != nil {
		log.Fatal(err)
	}

	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}
