package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword แปลงรหัสผ่านเป็นลายนิ้วมือ bcrypt ที่ย้อนกลับไม่ได้
func HashPassword(plain string) (string, error) {
    b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
    return string(b), err
}

// CheckPassword ตอบว่ารหัสที่พิมพ์มาตรงกับลายนิ้วมือที่เก็บไว้ไหม
func CheckPassword(hash, plain string) bool {
    return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoiYWRtaW4iLCJzdWIiOiIzIiwiZXhwIjoxNzkyNTg4NTcxLCJpYXQiOjE3OTI1ODQ5NzF9.piLWnzP8gI4TvbFg-ZBj2R5LweJkpdCvcc3IMML3byY