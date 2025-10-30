docker-down:
	@echo "🧹 Stopping and removing all containers, images, volumes..."
	docker-compose down -v --rmi all
	docker system prune -a --volumes -f

docker-check:
	@echo "🔍 Checking Docker state..."
	docker ps -a
	docker images
	docker volume ls
	docker network ls

docker-up:
	@echo "🚀 Launching Go application..."
	docker-compose up -d
	go run main.go

gofumpt:
	gofumpt -l -w .
