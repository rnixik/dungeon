const GameEventHandler = {
    FireballEvent(data) {
        this.projectiles.castPlayerFireball(data.clientId, data.x, data.y, data.direction, 500)
    },

    SwordAttackPrepareEvent(data) {

    },

    SwordAttackEvent(data) {
        const graphics = this.add.graphics();
        graphics.lineStyle(2, 0xff0000);

        graphics.beginPath();
        graphics.moveTo(data.x, data.y);
        graphics.lineTo(data.attackLineX, data.attackLineY);
        graphics.closePath();
        graphics.strokePath();

        this.time.delayedCall(200, () => {
            graphics.destroy();
        });
    },

    CreaturesStatsUpdateEvent(data) {
        for (const p of data.players) {
            const id = p.clientId;
            if (id === this.myClientId) {
                this.player.updateStatAndPosition(p);
                continue;
            }

            if (!this.players[id]) {
                this.players[id] = new Player(p.class, this, p)
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
        this.projectiles.shootMonsterArrow(data.monsterId, data.x1, data.y1, data.x2, data.y2, 400);
    },

    DemonFireballEvent(data) {
        this.projectiles.castMonsterFirebolt(data.monsterId, data.x1, data.y1, data.x2, data.y2, 700)
    },

    FireCircleEvent(data) {
        const numberOfProjectiles = 8;
        for (let i = 0; i < numberOfProjectiles; i++) {
            const angle = i * (Math.PI * 2) / numberOfProjectiles;
            const vector = new Phaser.Math.Vector2(Math.cos(angle), Math.sin(angle));
            this.projectiles.castMonsterFirespotVector(data.monsterId, data.x, data.y, vector, 200)
        }
    },

    DemonLightningEvent(data) {
        new LightingGroup(data.monsterId, data.x, data.y, this);
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
    },

    KeyCollectedEvent(data) {
        switch (data.number) {
            case "1": this.key1Collected = true; break;
            case "2": this.key2Collected = true; break;
            case "3": this.key3Collected = true; break;
            default: console.warn("unknown key number:", data.number); break;
        }

        this.addKeysIcons();
    },

    UpdateTilesEvent(data) {
        for (const t of data.tiles) {
            this.layerFloor.putTileAtWorldXY(t.tileId, t.x, t.y, false);
        }
    },

    SpawnSpikeEvent(data) {
        const s = this.add.sprite(data.x, data.y, 'spikes')
            .setOrigin(0, 0)
            .anims.play({ key: 'spikes', startFrame: Number(data.startFrame) });
        this.physics.add.existing(s);

        const safeFrames = [4, 5, 6, 7, 8];
        let canDamage = true;

        this.physics.add.overlap(s, this.player, (s, p) => {
            if (!canDamage) {
                return;
            }
            console.log('overlap with spikes', s.anims.currentFrame.index);
            if (!safeFrames.includes(s.anims.currentFrame.index) && !this.isDead) {
                canDamage = false;
                this.sendGameCommand('HitPlayerCommand', {
                    monsterId: -1,
                    targetClientId: this.myClientId
                });
                setTimeout(() => canDamage = true, 1000);
            }
        }, null, this);
    }
}
