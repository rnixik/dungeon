const GameEventHandler = {
    FireballEvent(data) {
        this.bullets.fireBullet(data.clientId, data.x, data.y, data.direction)
    },

    CreaturesStatsUpdateEvent(data) {
        for (const p of data.players) {
            const id = p.clientId;
            if (id === this.myClientId) {
                if (this.player.hp !== p.hp) {
                    this.player.hp = p.hp;
                    if (this.player.hpText) {
                        this.player.hpText.setText(p.hp + '/100');
                    }
                }
                continue;
            }
            let justSpawned = false;
            if (!this.players[id]) {
                // spawn new player
                const np = this.physics.add.sprite(p.x, p.y, 'player', 1).setScale(PLAYER_SCALE).setDepth(DEPTH_PLAYER);
                np.id = id;
                np.hp = p.hp;

                np.setTint(Math.random() * 0xffffff);

                np.hpText = this.add.text(p.x, p.y, p.hp + '/100', { font: '8px Arial', fill: '#ffffff' }).setOrigin(0.5, 1).setDepth(DEPTH_PLAYER + 1);

                this.players[id] = np;
                this.bullets.addPlayer(np);

                justSpawned = true;
                console.log('spawn player', id, Object.keys(this.players).length);
            }

            this.updatePlayerPos(p);

            const pSprite = this.players[id];
            if (pSprite.hp !== p.hp || justSpawned) {
                pSprite.hp = p.hp;
                if (pSprite.hpText) {
                    pSprite.hpText.setText(p.hp + '/100');
                }
                if (p.hp === 0) {
                    if (pSprite.hpText) {
                        pSprite.hpText.destroy();
                        pSprite.setTint(0xff3333);
                        // avoid late changes of damage effect
                        this.time.delayedCall(110, () => pSprite.setTint(0xff3333), [], this);
                        pSprite.setDepth(DEPTH_DEAD);
                        pSprite.disableBody();
                    }
                }
            }
        }

        for (const m of data.monsters) {
            const id = m.id;
            if (!this.monsters[id]) {
                // spawn new monster
                this.monsters[id] = Monster.SpawnNewMonster(this, m);
                this.bullets.addMonster(this.monsters[id]);
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

    DamageEvent(data) {
        const pId = data.targetPlayerId;
        const mId = data.targetMonsterId;
        if (pId === this.myClientId) {
            this.player.setTint(0xff0000);
            this.time.delayedCall(100, () => this.player.clearTint(), [], this);
        }
        for (const i in this.players) {
            const p = this.players[i];
            if (p.id === pId) {
                p.setTint(0xff0000);
                this.time.delayedCall(100, () => p.clearTint(), [], this);
            }
        }
        for (const i in this.monsters) {
            const m = this.monsters[i];
            if (m.id === mId) {
                m.setTint(0xff0000);
                this.time.delayedCall(100, () => m.clearTint(), [], this);
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
