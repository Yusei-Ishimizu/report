package service

import (
	"net/http"
	"strconv"
	"fmt"//必要？？？？？？？？？？？？？？？？？
	"github.com/gin-gonic/gin"
    "github.com/gin-contrib/sessions"
    //"github.com/jmoiron/sqlx"
	database "todolist.go/db"
)

func NewTaskForm(ctx *gin.Context) {
    ctx.HTML(http.StatusOK, "form_new_task.html", gin.H{"Title": "Task registration"})
}

// TaskList renders list of tasks in DB
func TaskList(ctx *gin.Context) {
    userID := sessions.Default(ctx).Get("user")

	// Get DB connection
	db, err := database.GetConnection()
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}

	// Get query parameter
	kw := ctx.Query("kw")
    is_done := ctx.Query("is_done")

    

	// Get tasks in DB
	var tasks []database.Task

    /*var b bool

    if not_is_done != ""{
        b, err = strconv.ParseBool(not_is_done)
        if err != nil {
            Error(http.StatusBadRequest, "somthing is error")(ctx)
            return
        } 
    }*/
      
    query := "SELECT id, title, created_at, is_done FROM tasks INNER JOIN ownership ON task_id = id WHERE user_id = ?"

	switch {
    case kw != "":
        if is_done == "not_is_done"{
            query = query + " AND is_done = false AND title LIKE ?"
        }else{
            query = query + " AND title LIKE ?"
        }
        err = db.Select(&tasks, query, userID, "%" + kw + "%")
    default:
        if is_done == "not_is_done"{
            query = query + " AND is_done = false"
        }else{
            //err = db.Select(&tasks, query, userID)
        }
        err = db.Select(&tasks, query, userID)
        
    }
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
    

	// Render tasks
	ctx.HTML(http.StatusOK, "task_list.html", gin.H{"Title": "Task list", "Tasks": tasks, "Kw": kw})
}

// ShowTask renders a task with given ID
func ShowTask(ctx *gin.Context) {
	// Get DB connection
	db, err := database.GetConnection()
	if err != nil {
		Error(http.StatusInternalServerError, err.Error())(ctx)
		return
	}

	// parse ID given as a parameter
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		Error(http.StatusBadRequest, err.Error())(ctx)
		return
	}

	// Get a task with given ID
	var task database.Task
	err = db.Get(&task, "SELECT * FROM tasks WHERE id=?", id) // Use DB#Get for one entry
	if err != nil {
		Error(http.StatusBadRequest, err.Error())(ctx)
		return
	}

	// Render task
	ctx.HTML(http.StatusOK, "task.html", task)  // Modify it!!
}

func RegisterTask(ctx *gin.Context) {
    userID := sessions.Default(ctx).Get("user")

    // Get task title
    title, exist := ctx.GetPostForm("title")
    if !exist {
        Error(http.StatusBadRequest, "No title is given")(ctx)
        return
    }

    // Get DB connection
    db, err := database.GetConnection()
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
    tx := db.MustBegin()
    result, err := tx.Exec("INSERT INTO tasks (title) VALUES (?)", title)
    if err != nil {
        tx.Rollback()
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
    taskID, err := result.LastInsertId()
    if err != nil {
        tx.Rollback()
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
    _, err = tx.Exec("INSERT INTO ownership (user_id, task_id) VALUES (?, ?)", userID, taskID)
    if err != nil {
        tx.Rollback()
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
    tx.Commit()
    ctx.Redirect(http.StatusFound, fmt.Sprintf("/task/%d", taskID))
}

func EditTaskForm(ctx *gin.Context) {
    // ID の取得
    id, err := strconv.Atoi(ctx.Param("id"))
    if err != nil {
        Error(http.StatusBadRequest, err.Error())(ctx)
        return
    }
    // Get DB connection
    db, err := database.GetConnection()
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
    // Get target task
    var task database.Task
    err = db.Get(&task, "SELECT * FROM tasks WHERE id=?", id)
    if err != nil {
        Error(http.StatusBadRequest, err.Error())(ctx)
        return
    }
    // Render edit form
    ctx.HTML(http.StatusOK, "form_edit_task.html",
        gin.H{"Title": fmt.Sprintf("Edit task %d", task.ID), "Task": task})
}


func UpdateTask(ctx *gin.Context){
    id, err := strconv.Atoi(ctx.Param("id"))
    if err != nil {
        Error(http.StatusBadRequest, err.Error())(ctx)
        return
    }

	title, exist := ctx.GetPostForm("title")
    if !exist {
        Error(http.StatusBadRequest, "No title is given")(ctx)
        return
    }

	is_done, exist := ctx.GetPostForm("is_done")
    if !exist {
        Error(http.StatusBadRequest, "No is_done is given")(ctx)
        return
    }


	db, err := database.GetConnection()
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }

	b, err := strconv.ParseBool(is_done)
    if err != nil {
        Error(http.StatusBadRequest, "somthing is error")(ctx)
        return
    }         

    _, err = db.Exec("UPDATE tasks SET title = ?, is_done = ? WHERE id = ?",
							title, b, id)
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
		//fmt.Println("here1 is error!!!!!!!!!!!!!!!!")
        return
    }

    path := fmt.Sprintf("/task/%d", id) 
    ctx.Redirect(http.StatusFound, path)

}

func DeleteTask(ctx *gin.Context) {
    // ID の取得
    id, err := strconv.Atoi(ctx.Param("id"))
    if err != nil {
        Error(http.StatusBadRequest, err.Error())(ctx)
        return
    }
    // Get DB connection
    db, err := database.GetConnection()
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
    // Delete the task from DB
    _, err = db.Exec("DELETE FROM tasks WHERE id=?", id)
    if err != nil {
        Error(http.StatusInternalServerError, err.Error())(ctx)
        return
    }
    // Redirect to /list
    ctx.Redirect(http.StatusFound, "/list")
}