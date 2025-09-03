up:
	stat docker-compose.yml > /dev/null 2>&1 || ln -s deployments/docker-compose.local.yml docker-compose.yml
	docker compose up -d --build
