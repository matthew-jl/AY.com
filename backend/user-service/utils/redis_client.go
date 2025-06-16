package utils

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

var Rdb *redis.Client

const VerificationCodeExpiry = 5 * time.Minute

func InitRedis() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		log.Println("UserSvc: REDIS_ADDR not set, Redis functionality for verification codes will be disabled.")
		return
	}

	Rdb = redis.NewClient(&redis.Options{Addr: redisAddr})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := Rdb.Ping(ctx).Result()
	if err != nil {
		log.Printf("UserSvc: Failed to connect to Redis at %s: %v. Verification codes might not use Redis.", redisAddr, err)
		Rdb = nil
		return
	}
	log.Println("UserSvc: Successfully connected to Redis.")
}

func StoreVerificationCode(ctx context.Context, email string, code string) error {
	if Rdb == nil {
		log.Println("UserSvc: Redis client not initialized. Skipping StoreVerificationCode.")
		return nil
	}
	key := fmt.Sprintf("verify_email:%s", email)
	err := Rdb.Set(ctx, key, code, VerificationCodeExpiry).Err()
	if err != nil {
		log.Printf("UserSvc: Error setting verification code in Redis for %s: %v", email, err)
		return err
	}
	log.Printf("UserSvc: Stored verification code for %s in Redis. Key: %s", email, key)
	return nil
}

func GetAndVerifyCodeFromRedis(ctx context.Context, email string, codeToVerify string) (bool, error) {
	if Rdb == nil {
		log.Println("UserSvc: Redis client not initialized. Skipping GetAndVerifyCodeFromRedis. Will rely on DB.")
		return false, errors.New("redis_unavailable")
	}
	key := fmt.Sprintf("verify_email:%s", email)
	storedCode, err := Rdb.Get(ctx, key).Result()

	if err == redis.Nil {
		log.Printf("UserSvc: Verification code for %s not found in Redis (expired or never set).", email)
		return false, nil
	} else if err != nil {
		log.Printf("UserSvc: Error getting verification code from Redis for %s: %v", email, err)
		return false, err
	}

	if storedCode == codeToVerify {
		errDel := Rdb.Del(ctx, key).Err()
		if errDel != nil {
			log.Printf("UserSvc: Warning - failed to delete verification code for %s from Redis after successful verification: %v", email, errDel)
		}
		log.Printf("UserSvc: Verification code for %s matched and deleted from Redis.", email)
		return true, nil
	}

	log.Printf("UserSvc: Verification code for %s did NOT match Redis. Expected: %s, Got: %s", email, storedCode, codeToVerify)
	return false, nil
}
