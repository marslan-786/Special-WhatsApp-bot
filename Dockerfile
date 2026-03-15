# Go 1.25 Alpine لیٹیسٹ ورژن
FROM golang:1.25-alpine

# GCC اور CGO کے ٹولز انسٹال کریں (SQLite کے لیے انتہائی ضروری)
RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

# آپ کا سارا کوڈ (main.go اور frontend فولڈر) کاپی کریں
COPY . .

# ڈوکر کے اندر ہی ماڈیول بنائیں اور لیٹیسٹ پیکجز انسٹال کریں
RUN go mod init silent-gateway && \
    go get go.mau.fi/whatsmeow@latest && \
    go get github.com/mattn/go-sqlite3@latest && \
    go get github.com/gorilla/websocket@latest && \
    go mod tidy

# CGO_ENABLED=1 کے ساتھ بلڈ کریں تاکہ SQLite صحیح کام کرے
RUN CGO_ENABLED=1 GOOS=linux go build -o bot .

# پورٹ ایکسپوز کریں
ENV PORT=8080
EXPOSE 8080

# کنٹینر رن ہونے پر بوٹ سٹارٹ کریں
CMD ["./bot"]
