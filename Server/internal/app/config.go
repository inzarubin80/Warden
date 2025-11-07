package app

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/inzarubin80/Warden/internal/app/defenitions"

	authinterface "github.com/inzarubin80/Warden/internal/app/authinterface"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/yandex"
)

type (
	Options struct {
		Addr string
	}
	path struct {
		index, getPoker, createPoker, createTask,
		getTasks, getTask, updateTask, deleteTask,
		getComents, addComent, setVotingTask,
		getVotingControlState, ws, login, session, refreshToken, logOut, getProviders,
		ping, vote, getUserEstimates, setVotingControlState, setUserName, getUser, setUserSettings, getLastSession, deletePoker string
	}

	sectrets struct {
		storeSecret        string
		accessTokenSecret  string
		refreshTokenSecret string
	}

	config struct {
		addr          string
		path          path
		sectrets      sectrets
		provadersConf authinterface.MapProviderOauthConf
	}
)

func NewConfig(opts Options) config {

	imageDataYandexAuth, err := os.ReadFile("images/yandex-auth.png")
	if err != nil {
		fmt.Errorf(err.Error())
	}

	imageBase64 := base64.StdEncoding.EncodeToString(imageDataYandexAuth)

	provaders := make(authinterface.MapProviderOauthConf)
	provaders["yandex"] = &authinterface.ProviderOauthConf{
		Oauth2Config: &oauth2.Config{
			ClientID:     os.Getenv("CLIENT_ID_YANDEX"),
			ClientSecret: os.Getenv("CLIENT_SECRET_YANDEX"),
			RedirectURL:  os.Getenv("APP_ROOT") + "/YandexAuthCallback",
			Scopes:       []string{"login:email", "login:info"},
			Endpoint:     yandex.Endpoint,
		},
		UrlUserData: "https://login.yandex.ru/info?format=json",
		ImageBase64: imageBase64,
	}

	config := config{
		addr: opts.Addr,
		path: path{
			index:        "",
			ping:         "GET /api/ping",
			createPoker:  "POST	/api/poker",
			getProviders: "GET /api/providers",

			login:           "POST	/api/user/login",
			setUserName:     "POST	/api/user/name",
			setUserSettings: "POST	/api/user/settings",

			getUser: "GET	/api/user",

			refreshToken: "POST	/api/user/refresh",
			session:      "GET		/api/user/session",
			logOut:       "GET		/api/user/logout",

			getLastSession: fmt.Sprintf("GET	/api/sessions/{%s}/{%s}", defenitions.Page, defenitions.PageSize),
		},

		sectrets: sectrets{
			storeSecret:        os.Getenv("STORE_SECRET"),
			accessTokenSecret:  os.Getenv("ACCESS_TOKEN_SECRET"),
			refreshTokenSecret: os.Getenv("REFRESH_TOKEN_SECRET"),
		},

		provadersConf: provaders,
	}

	return config
}
