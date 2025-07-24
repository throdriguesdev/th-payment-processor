package handlers

import (
	"net/http"
	"time"
	"payment-processors/models"
	"payment-processors/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type PaymentHandler struct {
	storage *storage.InMemoryStorage
}

func NewPaymentHandler(storage *storage.InMemoryStorage) *PaymentHandler {
	return &PaymentHandler{
		storage: storage,
	}
}
