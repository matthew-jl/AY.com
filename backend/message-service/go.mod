module github.com/Acad600-TPA/WEB-MJ-242/backend/message-service

go 1.23.3

require (
	github.com/Acad600-TPA/WEB-MJ-242/backend/user-service v0.0.0
	github.com/Acad600-TPA/WEB-MJ-242/backend/media-service v0.0.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/lib/pq v1.10.9
	github.com/sirupsen/logrus v1.9.3
	google.golang.org/grpc v1.73.0
	google.golang.org/protobuf v1.36.6
	gorm.io/driver/postgres v1.6.0
	gorm.io/gorm v1.30.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.6.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sync v0.12.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250324211829-b45e905df463 // indirect
)

replace github.com/Acad600-TPA/WEB-MJ-242/backend/user-service => ../user-service

replace github.com/Acad600-TPA/WEB-MJ-242/backend/media-service => ../media-service
