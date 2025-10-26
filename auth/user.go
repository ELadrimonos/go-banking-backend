package auth

import (
	"net/http"
)

func (env *Env) GetUserHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := GetUserIDFromContext(r)
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	db := &DB{env.DB}
	user, err := db.GetUserByID(userID)
	if err != nil || user == nil {
		RespondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	// Return only public user information
	publicUser := struct {
		DNI      string `json:"dni"`
		FullName string `json:"full_name"`
		Email    string `json:"email"`
	}{
		DNI:      user.DNI,
		FullName: user.FullName,
		Email:    user.Email,
	}

	JSON(w, http.StatusOK, publicUser)
}
