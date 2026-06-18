package response

import "github.com/gin-gonic/gin"

// Response é o envelope JSON padronizado de todas as respostas da API.
// Sucesso: {"data": <payload>, "error": null}
// Erro:    {"data": null, "error": {"code": "...", "message": "..."}}
type Response struct {
	Data  interface{} `json:"data"`
	Error *ErrorInfo  `json:"error"`
}

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Success escreve uma resposta de sucesso com o payload em "data" e "error": null.
func Success(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, Response{
		Data:  data,
		Error: nil,
	})
}

// Error escreve uma resposta de erro com "data": null e o código/mensagem em "error".
func Error(c *gin.Context, statusCode int, code string, message string) {
	c.JSON(statusCode, Response{
		Data: nil,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	})
}
