package service
 
import (
	"crypto/sha256"
    "encoding/hex"
    "net/http"
    "fmt" 
    //"strconv"
    "github.com/gin-gonic/gin"
    "github.com/gin-contrib/sessions"
    database "todolist.go/db"
)
 
func NewUserForm(ctx *gin.Context) {
    ctx.HTML(http.StatusOK, "new_user_form.html", gin.H{"Title": "Register user"})
}

func hash(pw string) []byte {
    const salt = "todolist.go#"
    h := sha256.New()
    h.Write([]byte(salt))
    h.Write([]byte(pw))
    return h.Sum(nil)
}

func RegisterUser(ctx *gin.Context) {
    // フォームデータの受け取り
    username := ctx.PostForm("username")
    password := ctx.PostForm("password")
	password_check := ctx.PostForm("password_check")
    switch {
    case username == "":
        ctx.HTML(http.StatusBadRequest, "new_user_form.html", gin.H{"Title": "Register user", "Error": "Username is not provided", "Username": username})
    case password == "":
        ctx.HTML(http.StatusBadRequest, "new_user_form.html", gin.H{"Title": "Register user", "Error": "Password is not provided", "Password": password})
    case password_check == "":
        ctx.HTML(http.StatusBadRequest, "new_user_form.html", gin.H{"Title": "Register user", "Error": "Password_Check is not provided", "Password_Check": password_check})
    }
    
    // DB 接続
    db, err := database.GetConnection()
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
 
    // 重複チェック
    var duplicate int
    err = db.Get(&duplicate, "SELECT COUNT(*) FROM users WHERE name=?", username)
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
    if duplicate > 0 {
        ctx.HTML(http.StatusBadRequest, "new_user_form.html", gin.H{"Title": "Register user", "Error": "Username is already taken", "Username": username, "Password": password})
        return
    }

	// PWタイプミス確認
	if password != password_check {
		ctx.HTML(http.StatusBadRequest, "new_user_form.html", gin.H{"Title": "Register user", "Error": "Typing Password is missed", "Username": username, "Password": password})
        return
	}

    // DB への保存
    result, err := db.Exec("INSERT INTO users(name, password) VALUES (?, ?)", username, hash(password))
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
 
    // 保存状態の確認
    id, _ := result.LastInsertId()
    var user database.User
    err = db.Get(&user, "SELECT id, name, password FROM users WHERE id = ?", id)
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
    //ctx.JSON(http.StatusOK, user)
    ctx.Redirect(http.StatusFound, "/login")
}

const userkey = "user"

func NewLoginForm(ctx *gin.Context) {
    ctx.HTML(http.StatusOK, "login.html", gin.H{"Title": "Login"})
}
 
func Login(ctx *gin.Context) {
    username := ctx.PostForm("username")
    password := ctx.PostForm("password")
 
    db, err := database.GetConnection()
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
 
    // ユーザの取得
    var user database.User
    err = db.Get(&user, "SELECT id, name, password FROM users WHERE name = ?", username)
    if err != nil {
        ctx.HTML(http.StatusBadRequest, "login.html", gin.H{"Title": "Login", "Username": username, "Error": "No such user"})
        return
    }
 
    // パスワードの照合
    if hex.EncodeToString(user.Password) != hex.EncodeToString(hash(password)) {
        ctx.HTML(http.StatusBadRequest, "login.html", gin.H{"Title": "Login", "Username": username, "Error": "Incorrect password"})
        return
    }
 
    // セッションの保存
    session := sessions.Default(ctx)
    session.Set(userkey, user.ID)
    session.Save()
 
    ctx.Redirect(http.StatusFound, "/list")
}

func LoginCheck(ctx *gin.Context) {
    if sessions.Default(ctx).Get(userkey) == nil {
        ctx.Redirect(http.StatusFound, "/login")
        ctx.Abort()
    } else {
        ctx.Next()
    }
}

func Logout(ctx *gin.Context) {
    session := sessions.Default(ctx)
    session.Clear()
    session.Options(sessions.Options{MaxAge: -1})
    session.Save()
    ctx.Redirect(http.StatusFound, "/")
}

func DeleteUser(ctx *gin.Context) {
    userID := sessions.Default(ctx).Get("user")

    // Get DB connection
    db, err := database.GetConnection()
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }

    tx := db.MustBegin()
    // Delete the task from DB
    _, err = tx.Exec("DELETE FROM users WHERE id=?", userID)
    if err != nil {
        tx.Rollback()
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }

    _, err = tx.Exec("DELETE FROM ownership WHERE user_id=?", userID)
    if err != nil {
        tx.Rollback()
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }

    tx.Commit()

    // Redirect to /list
    ctx.Redirect(http.StatusFound, "/")
}

func EditUser(ctx *gin.Context){
    userID := sessions.Default(ctx).Get("user")

    // Get DB connection
    db, err := database.GetConnection()
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }

    // Get target user
    var user database.User
    err = db.Get(&user, "SELECT * FROM users WHERE id=?", userID)
    if err != nil {
        Error(http.StatusBadRequest, err.Error())(ctx)
        return
    }

    // Render edit form
    ctx.HTML(http.StatusOK, "form_edit_user.html",
        gin.H{"Title": fmt.Sprintf("Edit user %d", user.Name), "User": user})
}

func UpdateUser(ctx *gin.Context){
    userID := sessions.Default(ctx).Get("user")

    username, exist := ctx.GetPostForm("new_name")
    if !exist {
        Error(http.StatusBadRequest, "No name is given")(ctx)
        return
    }

    password, exist := ctx.GetPostForm("new_password")
    if !exist {
        Error(http.StatusBadRequest, "No new password is given")(ctx)
        return
    }

    password_check, exist := ctx.GetPostForm("new_password_check")
    if !exist {
        Error(http.StatusBadRequest, "No check new password is given")(ctx)
        return
    }

    now_password, exist := ctx.GetPostForm("now_password")
    if !exist {
        Error(http.StatusBadRequest, "No now password is given")(ctx)
        return
    }

    db, err := database.GetConnection()
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }

    // ユーザの取得
    var user database.User
    err = db.Get(&user, "SELECT id, name, password FROM users WHERE name = ?", username)
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }

    // 重複チェック
    var duplicate int
    err = db.Get(&duplicate, "SELECT COUNT(*) FROM users WHERE name=? and NOT id = ?", username, userID)
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
    if duplicate > 0 {
        ctx.HTML(http.StatusBadRequest, "form_edit_user.html", gin.H{"Title": fmt.Sprintf("Edit user %d", user.Name), "User": user, "Error": "Username is already taken"})
        return
    }

	// PWタイプミス確認
	if password != password_check {
		ctx.HTML(http.StatusBadRequest, "form_edit_user.html", gin.H{"Title": fmt.Sprintf("Edit user %d", user.Name), "User": user, "Error": "Password Typemiss"})
        return
	}

    // パスワードの照合
    if hex.EncodeToString(user.Password) != hex.EncodeToString(hash(now_password)) {
        ctx.HTML(http.StatusBadRequest, "form_edit_user.html", gin.H{"Title": fmt.Sprintf("Edit user %d", user.Name), "User": user, "Error": "Incorect Password "})
        return
    }

    _, err = db.Exec("UPDATE users SET name = ?, password = ? WHERE id = ?",
                                        username, hash(password), userID)
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }

    ctx.Redirect(http.StatusFound, "/list")

    /*
	
       

    _, err = db.Exec("UPDATE tasks SET title = ?, is_done = ?, importance = ? WHERE id = ?",
							title, b, importance, id)
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }

    path := fmt.Sprintf("/task/%d", id) 
    ctx.Redirect(http.StatusFound, path)
    */
}