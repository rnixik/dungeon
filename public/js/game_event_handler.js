const GameEventHandler = {
    FireballEvent(data) {
        this.projectiles.castFireball(data.clientId, data.x, data.y, data.direction, 500)
    },

    CreaturesStatsUpdateEvent(data) {
        for (const p of data.players) {
            const id = p.clientId;
            if (id === this.myClientId) {
                this.player.updateStatAndPosition(p);
                continue;
            }

            if (!this.players[id]) {
                this.players[id] = new Player("mage", this, p)
                this.projectiles.addPlayer(this.players[id]);
            }

            this.players[id].updateStatAndPosition(p);
        }

        for (const m of data.monsters) {
            const id = m.id;
            if (!this.monsters[id]) {
                this.monsters[id] = Monster.SpawnNewMonster(this, m);
                this.projectiles.addMonster(this.monsters[id]);
            }

            this.monsters[id].updateStatAndPosition(m);
        }
    },

    CreaturesPosUpdateEvent(data) {
        for (const p of data.players) {
            this.updatePlayerPos(p)
        }

        for (const m of data.monsters) {
            this.updateMonsterPos(m);
        }
    },

    PlayerDeathEvent(data) {
        if (data.clientId === this.myClientId) {
            this.add.text(this.scale.width / 2, this.scale.height / 2, 'YOU DIED', { font: '24px Arial', fill: '#ff0000' })
                .setOrigin(0.5, 0.5)
                .setScrollFactor(0, 0)
                .setDepth(DEPTH_UI);
            this.isDead = true;
            console.log('this is my death');
        }
    },

    ArrowEvent(data) {
        const ar = this.physics.add.sprite(data.x1, data.y1, 'arrow', 0).setScale(2);
        const dir = new Phaser.Math.Vector2(data.x2 - data.x1, data.y2 - data.y1).normalize();
        const ang = Phaser.Math.Angle.Between(data.x1, data.y1, data.x2, data.y2);
        ar.setOrigin(0.5, 0.5).setRotation(ang).setAngle(ar.angle - 90);
        ar.setVelocity(dir.x * 400, dir.y * 400);
        this.physics.add.collider(ar, this.layerWalls, () => ar.destroy(), null, this);
        this.physics.add.overlap(ar, this.player, () => {
            ar.destroy();
            this.sendGameCommand('HitPlayerCommand', {
                monsterId: data.monsterId,
                targetClientId: this.myClientId
            });
        }, null, this);
    },

    DemonFireballEvent(data) {
        const ar = this.physics.add.sprite(data.x1, data.y1, 'bullet', 0).setScale(1);
        const dir = new Phaser.Math.Vector2(data.x2 - data.x1, data.y2 - data.y1).normalize();
        ar.setVelocity(dir.x * 700, dir.y * 700);
        this.physics.add.collider(ar, this.layerWalls, () => ar.destroy(), null, this);
        this.physics.add.overlap(ar, this.player, () => {
            ar.destroy();
            this.sendGameCommand('HitPlayerCommand', {
                monsterId: data.monsterId,
                targetClientId: this.myClientId
            });
        }, null, this);
    },

    FireCircleEvent(data) {
        for (let i = 0; i < 8; i++) {
            const angle = Phaser.Math.DegToRad(i * 45);
            const dir = new Phaser.Math.Vector2(Math.cos(angle), Math.sin(angle));
            const ar = this.physics.add.sprite(data.x, data.y, 'bullet', 0).setScale(1);
            ar.setVelocity(dir.x * 300, dir.y * 300);
            this.physics.add.collider(ar, this.layerWalls, () => ar.destroy(), null, this);
            this.physics.add.overlap(ar, this.player, () => {
                ar.destroy();
                this.sendGameCommand('HitPlayerCommand', {
                    monsterId: data.monsterId,
                    targetClientId: this.myClientId
                });
            }, null, this);
        }
    },

    DamageEvent(data) {
        const pId = data.targetPlayerId;
        const mId = data.targetMonsterId;
        if (pId === this.myClientId) {
            this.player.takeDamage(data.damage)
        }
        for (const i in this.players) {
            const p = this.players[i];
            if (p.id === pId) {
                p.takeDamage(data.damage);
            }
        }
        for (const i in this.monsters) {
            const m = this.monsters[i];
            if (m.id === mId) {
                m.takeDamage(data.damage);
            }
        }
    },

    ChestOpenEvent(data) {
        const objId = data.objectId;
        const obj = this.gameObjects[objId];
        if (obj && obj instanceof Chest) {
            obj.open();
        } else {
            console.warn("no such chest to open:", objId);
        }
    }
}
