package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
	ldap "gopkg.in/ldap.v3"
)

func newAdminTemplate() *template.Template {
	funcMap := template.FuncMap{
		"formatTime": formatTime,
	}

	return template.Must(template.New("adm").Funcs(funcMap).ParseGlob("templates/adm/*.html"))
}

var (
	admT = newAdminTemplate()
)

func adminLogin(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	session, _ := store.Get(r, "admin")
	if session.Values["admin_logged"] == "true" {
		http.Redirect(w, r, "/adm", http.StatusFound)
		return
	}
	admT.ExecuteTemplate(w, "login", nil)
}

func authAdmin(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	session, _ := store.Get(r, "admin")
	if session.Values["admin_logged"] == "true" {
		http.Redirect(w, r, "/adm", http.StatusFound)
		return
	}

	login := r.FormValue("login")
	searchRequest := ldap.NewSearchRequest(
		"dc=mc,dc=asu,dc=ru",
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&(memberOf=cn=billing,ou=groups,ou=vc,dc=mc,dc=asu,dc=ru)(samAccountName=%s))", login),
		[]string{"dn"},
		nil,
	)

	err := ldapAuth(w, r, searchRequest)
	if err != nil {
		log.Println(err)
		url := fmt.Sprint("/admin-login?err=1")
		http.Redirect(w, r, url, http.StatusFound)
		return
	}

	session.Values["admin_logged"] = "true"
	session.Save(r, w)
	http.Redirect(w, r, "/adm", http.StatusFound)
}

func adminIndex(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	t := r.FormValue("type")
	name := r.FormValue("name")
	account := r.FormValue("account")

	var users []User
	var err error
	if name != "" {
		users, err = getUsersByName(name)
		if err != nil {
			log.Printf("could not get users by name=%v: %v", name, err)
		}
	} else if account != "" {
		users, err = getUsersByAccount(account)
		if err != nil {
			log.Printf("could not get users by account=%v: %v", account, err)
		}
	} else {
		users, err = getUsersByType(t)
		if err != nil {
			log.Printf("could not get users by type=%v: %v", t, err)
		}
	}
	if err != nil {
		http.Error(w, "Что-то пошло не так", http.StatusInternalServerError)
		return
	}

	admT.ExecuteTemplate(w, "index", users)
}

func adminLogout(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	session, _ := store.Get(r, "admin")
	session.Values["admin_logged"] = "false"
	session.Save(r, w)
	http.Redirect(w, r, "/admin-login", http.StatusFound)
}

func userInfo(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	id, _ := strconv.Atoi(r.FormValue("id"))

	user, err := getUserByID(id)
	if err != nil {
		log.Printf("could not get user by id=%v: %v", id, err)
		http.Error(w, "Что-то пошло не так", http.StatusInternalServerError)
		return
	}

	admT.ExecuteTemplate(w, "user-info", user)
}

func userEditForm(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	id, _ := strconv.Atoi(r.FormValue("id"))

	user, err := getUserByID(id)
	if err != nil {
		log.Printf("could not get user by id=%v: %v", id, err)
		http.Error(w, "Что-то пошло не так", http.StatusInternalServerError)
		return
	}

	admT.ExecuteTemplate(w, "edit-user-form", user)
}

func editUser(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	id, _ := strconv.Atoi(r.FormValue("id"))
	name := r.FormValue("name")
	agreement := r.FormValue("agreement")
	login := r.FormValue("login")
	tariff := r.FormValue("tariff")
	phone := r.FormValue("phone")
	comment := r.FormValue("comment")
	connectionPlace := r.FormValue("connectionPlace")

	user := User{
		ID:              id,
		Name:            name,
		Agreement:       agreement,
		Login:           login,
		Tariff:          tariffFromString(tariff),
		Phone:           phone,
		Comment:         comment,
		ConnectionPlace: connectionPlace,
	}

	err := updateUser(user)
	if err != nil {
		log.Printf("could not update user with id=%v: %v", id, err)
		http.Error(w, "Что-то пошло не так", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/adm", http.StatusFound)
}

func newUserForm(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	admT.ExecuteTemplate(w, "new-user-form", nil)
}

func addNewUser(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	name := r.FormValue("name")
	agreement := r.FormValue("agreement")
	login := r.FormValue("login") + "@stud.asu.ru"
	tariff := r.FormValue("tariff")
	phone := r.FormValue("phone")
	comment := r.FormValue("comment")
	connectionPlace := r.FormValue("connectionPlace")

	moneyStr := r.FormValue("money")
	money := 0
	if moneyStr != "" {
		money, _ = strconv.Atoi(moneyStr)
	}

	user := User{
		Name:            name,
		Agreement:       agreement,
		Login:           login,
		Tariff:          tariffFromString(tariff),
		Phone:           phone,
		Comment:         comment,
		ConnectionPlace: connectionPlace,
		Money:           money,
	}

	id, err := addUserToDB(user)
	if err != nil {
		log.Printf("could not add user to mysql with id=%v: %v", id, err)
		http.Error(w, "Что-то пошло не так", http.StatusInternalServerError)
		return
	}

	if money != 0 {
		err = addPaymentInfo(id, money)
		if err != nil {
			log.Printf("could not add payment info about user with id=%v: %v", id, err)
			http.Error(w, "Что-то пошло не так", http.StatusInternalServerError)
			return
		}
	}

	err = withdrawMoney(id)
	if err != nil {
		log.Printf("could not withdraw money from user with id=%v: %v", id, err)
		http.Error(w, "Что-то пошло не так", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/adm", http.StatusFound)
}

func usersStatistics(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	activeUsersCount, err := getCountOfActiveUsers()
	if err != nil {
		log.Printf("could not get count of active users: %v", err)
		http.Error(w, "Что-то пошло не так", http.StatusInternalServerError)
		return
	}

	inactiveUsersCount, err := getCountOfInactiveUsers()
	if err != nil {
		log.Printf("could not get count of inactive users: %v", err)
		http.Error(w, "Что-то пошло не так", http.StatusInternalServerError)
		return
	}

	allMoney, err := getAllMoneyWeHave()
	if err != nil {
		log.Printf("could not get sum of all money we have: %v", err)
		http.Error(w, "Что-то пошло не так", http.StatusInternalServerError)
		return
	}

	J := struct {
		ActiveUsersCount   int `json:"active_users_count"`
		InactiveUsersCount int `json:"inactive_users_count"`
		AllMoney           int `json:"all_money"`
	}{
		ActiveUsersCount:   activeUsersCount,
		InactiveUsersCount: inactiveUsersCount,
		AllMoney:           allMoney,
	}
	json.NewEncoder(w).Encode(&J)
}

func deleteUser(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	id, _ := strconv.Atoi(r.FormValue("id"))
	err := deleteUserByID(id)
	if err != nil {
		log.Printf("could not delete user from mysql with id=%v: %v", id, err)
		http.Error(w, "Что-то пошло не так", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/adm", http.StatusFound)
}

func payForm(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	id, _ := strconv.Atoi(r.FormValue("id"))

	user, err := getUserByID(id)
	if err != nil {
		log.Printf("could not get user by id=%v: %v", id, err)
		http.Error(w, "Что-то пошло не так", http.StatusInternalServerError)
		return
	}

	admT.ExecuteTemplate(w, "payment", user)
}

func pay(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	id, _ := strconv.Atoi(r.FormValue("id"))
	moneyStr := r.FormValue("money")
	money, _ := strconv.Atoi(moneyStr)

	err := addMoney(id, money)
	if err != nil {
		log.Printf("could not add money to user with id=%v: %v", id, err)
		http.Error(w, "Что-то пошло не так", http.StatusInternalServerError)
		return
	}

	err = addPaymentInfo(id, money)
	if err != nil {
		log.Printf("could not add payment info about user with id=%v: %v", id, err)
		http.Error(w, "Что-то пошло не так", http.StatusInternalServerError)
		return
	}

	err = withdrawMoney(id)
	if err != nil {
		log.Printf("could not withdraw money from user with id=%v: %v", id, err)
		http.Error(w, "Что-то пошло не так", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/adm", http.StatusFound)
}

func tariffFromString(s string) (t Tariff) {
	pieces := strings.Split(s, " ")
	t.ID, _ = strconv.Atoi(pieces[0])
	t.Name = pieces[1]
	t.Price, _ = strconv.Atoi(pieces[2])
	return t
}
