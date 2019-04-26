package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/sessions"
	"github.com/julienschmidt/httprouter"
)

var (
	store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))
)

func init() {
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
	}
}

func main() {
	router := httprouter.New()

	router.ServeFiles("/assets/*filepath", http.Dir("assets/"))

	router.GET("/admin-login", accessLog(adminLogin))
	router.GET("/adm", accessLog(adminAuthCheck(adminIndex)))
	router.GET("/admin-logout", accessLog(adminLogout))
	router.GET("/user-logout", accessLog(userLogout))
	router.GET("/user-login", accessLog(userLogin))
	router.GET("/add-user", accessLog(adminAuthCheck(newUserForm)))
	router.GET("/user-info", accessLog(adminAuthCheck(userInfo)))
	router.GET("/edit-user", accessLog(adminAuthCheck(userEditForm)))
	router.GET("/delete-user", accessLog(adminAuthCheck(deleteUser)))
	router.GET("/", accessLog(userAuthCheck(userIndex)))
	router.GET("/pay", accessLog(adminAuthCheck(payForm)))
	router.GET("/settings", accessLog(userAuthCheck(userSettings)))
	router.GET("/confirm-settings", accessLog(userAuthCheck(confirmSettings)))

	router.POST("/admin-login", accessLog(authAdmin))
	router.POST("/user-login", accessLog(authUser))
	router.POST("/add-user", accessLog(adminAuthCheck(addNewUser)))
	router.POST("/edit-user", accessLog(adminAuthCheck(editUser)))
	router.POST("/pay", accessLog(adminAuthCheck(pay)))
	router.POST("/settings", accessLog(userAuthCheck(editUserSettings)))

	go func() {
		for {
			time.Sleep(1 * time.Minute)
			users, err := checkPaymentNeed()
			if err != nil {
				log.Println("Cannot check payment need.", err)
				continue
			}

			if len(users) == 0 {
				continue
			}

			err = sendPaymentNotification(users)
			if err != nil {
				log.Println("Cannot send payment notification.", err)
			}
		}
	}()

	err := http.ListenAndServe(":8081", router)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
