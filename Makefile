up:
	echo -n "1">./version
	docker compose -p dungeon -f deployments/docker-compose.local.yml up -d --build

stop:
	docker compose -p dungeon -f deployments/docker-compose.local.yml stop