package web

import (
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

func getuser(r *http.Request) string {
	user := ""
	//check cookie
	if c, err := r.Cookie("user"); err == nil {
		user = c.Value
	}
	return user
}

func UserHandler(f func(http.ResponseWriter,
	*http.Request, string)) func(http.ResponseWriter,
	*http.Request) {
	return func(w http.ResponseWriter,
		r *http.Request) {
		user := getuser(r)
		if user == "" {
			http.Redirect(w, r, "/main", http.StatusTemporaryRedirect)
			return
		}
		f(w, r, user)
	}
}

func usersPage(w http.ResponseWriter, r *http.Request, user string) {
	data := struct {
		User string
	}{
		User: user,
	}
	renderTemplate(w, "users", data)
}

func deleteUserPage(w http.ResponseWriter, r *http.Request) {
	os.RemoveAll(userDirectory(mux.Vars(r)["user"]))
	http.Redirect(w, r, "/users", http.StatusTemporaryRedirect)
}
