package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could read thumbnail", err)
		return
	}
	mediaType := header.Header.Get("Content-Type")
	mt, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't load file", err)
		return
	}
	log.Print(mt)

	var ext string
	switch mt {
	case "image/jpeg":
		ext = "jpg"
	case "image/png":
		ext = "png"
	default:
		respondWithError(w, http.StatusBadRequest, "invalid file format", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not get video data", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	b := make([]byte, 32)
	_, err = rand.Read(b)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error saving thumbnail", err)
		return
	}
	randStr := base64.RawURLEncoding.EncodeToString(b)

	url := fmt.Sprintf("http://localhost:%s/assets/%s.%s", cfg.port, randStr, ext)
	video.ThumbnailURL = &url

	filePath := filepath.Join(cfg.assetsRoot, randStr+"."+ext)
	diskFile, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error saving thumbnail", err)
		return
	}
	if _, err = io.Copy(diskFile, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "error saving thumbnail", err)
		return
	}
	defer diskFile.Close()

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldn't update video metadata", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
