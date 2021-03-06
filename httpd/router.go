package httpd

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"

	"github.com/drakkan/sftpgo/dataprovider"
	"github.com/drakkan/sftpgo/logger"
	"github.com/drakkan/sftpgo/metrics"
	"github.com/drakkan/sftpgo/sftpd"
	"github.com/drakkan/sftpgo/utils"
)

// GetHTTPRouter returns the configured HTTP handler
func GetHTTPRouter() http.Handler {
	return router
}

func initializeRouter(staticFilesPath string, enableProfiler, enableWebAdmin bool) {
	router = chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(logger.NewStructuredLogger(logger.GetLogger()))
	router.Use(middleware.Recoverer)

	if enableProfiler {
		logger.InfoToConsole("enabling the built-in profiler")
		logger.Info(logSender, "", "enabling the built-in profiler")
		router.Mount(pprofBasePath, middleware.Profiler())
	}

	router.NotFound(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sendAPIResponse(w, r, nil, "Not Found", http.StatusNotFound)
	}))

	router.MethodNotAllowed(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sendAPIResponse(w, r, nil, "Method not allowed", http.StatusMethodNotAllowed)
	}))

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, webUsersPath, http.StatusMovedPermanently)
	})

	router.Group(func(router chi.Router) {
		router.Use(checkAuth)

		router.Get(webBasePath, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, webUsersPath, http.StatusMovedPermanently)
		})

		metrics.AddMetricsEndpoint(metricsPath, router)

		router.Get(versionPath, func(w http.ResponseWriter, r *http.Request) {
			render.JSON(w, r, utils.GetAppVersion())
		})

		router.Get(providerStatusPath, func(w http.ResponseWriter, r *http.Request) {
			err := dataprovider.GetProviderStatus(dataProvider)
			if err != nil {
				sendAPIResponse(w, r, err, "", http.StatusInternalServerError)
			} else {
				sendAPIResponse(w, r, err, "Alive", http.StatusOK)
			}
		})

		router.Get(activeConnectionsPath, func(w http.ResponseWriter, r *http.Request) {
			render.JSON(w, r, sftpd.GetConnectionsStats())
		})

		router.Delete(activeConnectionsPath+"/{connectionID}", handleCloseConnection)
		router.Get(quotaScanPath, getQuotaScans)
		router.Post(quotaScanPath, startQuotaScan)
		router.Get(quotaScanVFolderPath, getVFolderQuotaScans)
		router.Post(quotaScanVFolderPath, startVFolderQuotaScan)
		router.Get(userPath, getUsers)
		router.Post(userPath, addUser)
		router.Get(userPath+"/{userID}", getUserByID)
		router.Put(userPath+"/{userID}", updateUser)
		router.Delete(userPath+"/{userID}", deleteUser)
		router.Get(folderPath, getFolders)
		router.Post(folderPath, addFolder)
		router.Delete(folderPath, deleteFolderByPath)
		router.Get(dumpDataPath, dumpData)
		router.Get(loadDataPath, loadData)
		if enableWebAdmin {
			router.Get(webUsersPath, handleGetWebUsers)
			router.Get(webUserPath, handleWebAddUserGet)
			router.Get(webUserPath+"/{userID}", handleWebUpdateUserGet)
			router.Post(webUserPath, handleWebAddUserPost)
			router.Post(webUserPath+"/{userID}", handleWebUpdateUserPost)
			router.Get(webConnectionsPath, handleWebGetConnections)
			router.Get(webFoldersPath, handleWebGetFolders)
			router.Get(webFolderPath, handleWebAddFolderGet)
			router.Post(webFolderPath, handleWebAddFolderPost)
		}
	})

	if enableWebAdmin {
		router.Group(func(router chi.Router) {
			compressor := middleware.NewCompressor(5)
			router.Use(compressor.Handler)
			fileServer(router, webStaticFilesPath, http.Dir(staticFilesPath))
		})
	}
}

func handleCloseConnection(w http.ResponseWriter, r *http.Request) {
	connectionID := chi.URLParam(r, "connectionID")
	if connectionID == "" {
		sendAPIResponse(w, r, nil, "connectionID is mandatory", http.StatusBadRequest)
		return
	}
	if sftpd.CloseActiveConnection(connectionID) {
		sendAPIResponse(w, r, nil, "Connection closed", http.StatusOK)
	} else {
		sendAPIResponse(w, r, nil, "Not Found", http.StatusNotFound)
	}
}

func fileServer(r chi.Router, path string, root http.FileSystem) {
	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}
