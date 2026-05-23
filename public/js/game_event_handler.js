const DEBUG_TRAPS = false; // Set to true to show trap debug rectangles

const GameEventHandler = {
    FireballEvent(data) {
        this.projectiles.castPlayerFireball(data.clientId, data.x, data.y + 10, data.direction, 500, data.distance)
    },

    ShootArrowEvent(data) {
        this.projectiles.shootPlayerArrow(data.clientId, data.x1, data.y1, data.x2, data.y2, data.velocity)
    },

    SwordAttackPrepareEvent(data) {

    },

    SwordAttackEvent(data) {
        // const graphics = this.add.graphics();
        // graphics.lineStyle(2, 0xff0000);
        //
        // graphics.beginPath();
        // graphics.moveTo(data.x, data.y);
        // graphics.lineTo(data.attackLineX, data.attackLineY);
        // graphics.closePath();
        // graphics.strokePath();
        //
        // graphics.strokeRect(data.x - data.radius, data.y - data.radius, data.radius * 2, data.radius * 2);
        // graphics.strokeRect(data.attackLineX - data.radius, data.attackLineY - data.radius, data.radius * 2, data.radius * 2);
        //
        // this.time.delayedCall(200, () => {
        //     graphics.destroy();
        // });

        const scale = data.radius / 32; // assuming original sprite size is 32x32

        const attackSprite1 = this.add.sprite(data.x-20, data.y, 'melee_surround')
            .setScale(scale)
            .setDepth(DEPTH_PLAYER - 0.1)
            .setOrigin(0.5, 0.5)
            .setMask(this.mask);
        attackSprite1.anims.play('melee_surround', true);
        attackSprite1.on('animationcomplete', () => {
            attackSprite1.destroy();
        });
        const attackSprite2 = this.add.sprite(data.attackLineX, data.attackLineY, 'melee_attack')
            .setScale(scale)
            .setDepth(DEPTH_MONSTER + 0.1)
            .setOrigin(0.5, 0.5)
            .setMask(this.mask);
        attackSprite2.anims.play('melee_attack', true);
        attackSprite2.on('animationcomplete', () => {
            attackSprite2.destroy();
        });

        this.cameras?.main?.shake(80, 0.01);


        if (data.attackLineX - data.x > 0) {
            // right
            attackSprite1.x = data.x + 20;
        }
        if (data.attackLineX - data.x < 0) {
            // left
            attackSprite1.flipX = true;
            attackSprite2.flipX = true;
            attackSprite1.x = data.x - 20;
        }
        if (data.attackLineY - data.y < 0) {
            // up
            attackSprite1.setAngle(-90);
            attackSprite2.setAngle(-90);
            attackSprite1.y = data.y - 20;
        }
        if (data.attackLineY - data.y > 0) {
            // down
            attackSprite1.setAngle(90);
            attackSprite2.setAngle(90);
            attackSprite1.y = data.y + 20;
        }
    },

    CreaturesStatsUpdateEvent(data) {
        for (const p of data.players) {
            const id = p.clientId;
            if (id === this.myClientId) {
                this.player.updateStatAndPosition(p);
                this.player.setDisplayAlpha(p.isInvisible ? 0.3 : 1);
                continue;
            }

            if (!this.players[id]) {
                this.players[id] = new Player(p.class, this, p)
                this.projectiles.addPlayer(this.players[id]);
            }

            this.players[id].updateStatAndPosition(p);
            this.players[id].setDisplayAlpha(p.isInvisible ? 0 : 1);
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
            this.deadText = this.add.text(this.scale.width / 2, this.scale.height / 2, 'YOU DIED', { font: '24px Arial', fill: '#ff0000' })
                .setOrigin(0.5, 0.5)
                .setScrollFactor(0, 0)
                .setDepth(DEPTH_UI);
            this.isDead = true;

            this.respawnButton = this.add.text(this.scale.width / 2, this.scale.height / 2 + 40, 'RESPAWN', { font: '12px Arial', fill: '#f3c800' })
                .setOrigin(0.5, 0.5)
                .setScrollFactor(0, 0)
                .setDepth(DEPTH_UI).setInteractive({ useHandCursor: true }).on('pointerdown', () => {
                    this.sendGameCommand('RespawnCommand');
                });

            return;
        }

        const deadText = this.add.text(10, 10, data.nickname + ' dead', { font: '12px Arial', fill: '#ff0000' })
            .setOrigin(0, 0)
            .setScrollFactor(0, 0)
            .setDepth(DEPTH_UI);
        this.time.delayedCall(5000, () => {
            deadText.destroy();
        });
    },

    PlayerRespawnEvent(data) {
        if (data.clientId === this.myClientId) {
            this.player.respawn(data.x, data.y)
            this.isDead = false;
            this.deadText.destroy();
            this.respawnButton.destroy();
        } else {
            const p = this.players[data.clientId];
            if (p) {
                p.respawn(data.x, data.y);
            }
        }
    },

    ArrowEvent(data) {
        this.projectiles.shootMonsterArrow(data.monsterId, data.x1, data.y1, data.x2, data.y2, 400);
    },

    DemonFireballEvent(data) {
        this.projectiles.castMonsterFirebolt(data.monsterId, data.x1, data.y1, data.x2, data.y2, 700)
    },

    FireCircleEvent(data) {
        const numberOfProjectiles = 16;
        for (let i = 0; i < numberOfProjectiles; i++) {
            const angle = i * (Math.PI * 2) / numberOfProjectiles;
            const vector = new Phaser.Math.Vector2(Math.cos(angle), Math.sin(angle));
            this.projectiles.castMonsterFirespotVector(data.monsterId, data.x, data.y, vector, 200)
        }
    },

    DemonLightningEvent(data) {
        new DemonLightningGroup(data.monsterId, data.x, data.y, data.targetX, data.targetY, this);
    },

    SpiderWebEvent(data) {
        this.spawnWebArea(data.x, data.y);
    },

    DemonMageShieldEvent(data) {
        const m = this.monsters[data.targetId];
        if (m) {
            m.showShield(data.duration);
        }
        this.showSpellBeam(data.casterId, data.targetId, 0x4499ff);
    },

    DemonMageSpeedBoostEvent(data) {
        const m = this.monsters[data.targetId];
        if (m) {
            m.showSpeedBoost(data.duration);
        }
        this.showSpellBeam(data.casterId, data.targetId, 0xffcc00);
    },

    showSpellBeam(casterMonId, targetMonId, color) {
        const caster = this.monsters[casterMonId];
        const target = this.monsters[targetMonId];
        if (!caster || !target) return;

        const g = this.add.graphics();
        g.lineStyle(2, color, 1);
        g.lineBetween(caster.x, caster.y, target.x, target.y);
        g.setDepth(DEPTH_PROJECTILES);
        g.setMask(this.mask);

        this.tweens.add({
            targets: g,
            alpha: 0,
            duration: 500,
            ease: 'Linear',
            onComplete: () => g.destroy()
        });
    },

    spawnWebArea(x, y) {
        const halfSize = 24; // 1.5 tiles (3x3 tiles total = 48px)
        const size = halfSize * 2;
        const duration = 5000;

        const g = this.add.graphics();
        g.fillStyle(0x888888, 0.45);
        g.fillRect(x - halfSize, y - halfSize, size, size);
        g.lineStyle(1, 0xcccccc, 0.8);
        // Cross lines
        g.lineBetween(x - halfSize, y, x + halfSize, y);
        g.lineBetween(x, y - halfSize, x, y + halfSize);
        // Diagonal lines
        g.lineBetween(x - halfSize, y - halfSize, x + halfSize, y + halfSize);
        g.lineBetween(x + halfSize, y - halfSize, x - halfSize, y + halfSize);
        // Concentric rings
        g.strokeRect(x - halfSize / 2, y - halfSize / 2, halfSize, halfSize);
        g.setDepth(DEPTH_PROJECTILES - 1);
        g.setMask(this.mask);

        this.activeWebs = this.activeWebs || [];
        const webEntry = { x, y, halfSize, expiresAt: Date.now() + duration, graphics: g };
        this.activeWebs.push(webEntry);

        this.tweens.add({
            targets: g,
            alpha: 0,
            duration: duration,
            ease: 'Linear',
            onComplete: () => {
                g.destroy();
                this.activeWebs = this.activeWebs.filter(w => w !== webEntry);
            }
        });
    },

    GolemSlamEvent(data) {
        this.cameras?.main?.shake(250, 0.025);
        const g = this.add.graphics();
        g.x = data.x;
        g.y = data.y + 30;
        g.lineStyle(4, 0xff8800, 1);
        g.strokeCircle(0, 0, data.radius);
        g.setDepth(DEPTH_PROJECTILES);
        g.setMask(this.mask);
        this.tweens.add({
            targets: g,
            alpha: 0,
            scaleX: 1.5,
            scaleY: 1.5,
            duration: 500,
            ease: 'Cubic.easeOut',
            onComplete: () => g.destroy()
        });
    },

    DamageEvent(data) {
        const pId = data.targetPlayerId;
        const mId = data.targetMonsterId;
        
        // Show floating damage text
        if (data.x && data.y && data.damage) {
            this.showDamageText(data.x, data.y, data.damage);
        }
        
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

    showDamageText(x, y, damage) {
        const damageText = this.add.text(x, y - 10, `-${damage}`, {
            font: '16px Arial',
            fill: '#ff0000',
            stroke: '#000000',
            strokeThickness: 3
        })
        .setDepth(DEPTH_UI)
        .setOrigin(0.5, 0.5);

        // Animate the text floating upwards and fading
        this.tweens.add({
            targets: damageText,
            y: y - 50,
            alpha: 0,
            duration: 800,
            ease: 'Cubic.easeOut',
            onComplete: () => {
                damageText.destroy();
            }
        });
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
        // Legacy support - convert old spike events to new trap system
        const trapId = `spike_${data.x}_${data.y}`;
        this.createTrapSprite(trapId, data.x, data.y, Number(data.startFrame));
    },

    TrapStateChangedEvent(data) {
        const trapId = data.trapId;
        let trap = this.traps[trapId];

        if (!trap) {
            // Create trap sprite if it doesn't exist
            trap = this.createTrapSprite(trapId, data.x, data.y, data.frame);
        }
        
        // Update trap state and animation
        trap.state = data.state;
        
        // Frame mapping (updated):
        // 0: Active peak (fully extended)
        // 1-4: Cooldown (retracting)
        // 5: Armed (hidden)
        // 7-11: Active start (rising animation)
        
        // Update debug rectangle color based on state
        if (DEBUG_TRAPS && trap.debugGraphics) {
            trap.debugGraphics.clear();
            let debugColor = 0xff6600; // Default orange
            
            switch (data.state) {
                case 'armed':
                    debugColor = 0x00ff00; // Green - safe
                    break;
                case 'active':
                    debugColor = 0xff0000; // Red - dangerous!
                    break;
                case 'cooldown':
                    debugColor = 0x00aaff; // Blue - cooling down
                    break;
            }
            
            trap.debugGraphics.lineStyle(2, debugColor);
            trap.debugGraphics.strokeRect(trap.x, trap.y, 32, 32);
        }
        
        // Update sprite frame based on server data
        trap.sprite.setFrame(data.frame);
        trap.sprite.clearTint();
    },

    createTrapSprite(trapId, x, y, startFrame) {
        const s = this.add.sprite(x, y, 'spikes')
            .setOrigin(0, 0)
            .setFrame(startFrame || 0);
        this.physics.add.existing(s);
        
        // Debug rectangle to show trap area
        let debugGraphics = null;
        if (DEBUG_TRAPS) {
            debugGraphics = this.add.graphics();
            debugGraphics.lineStyle(2, 0xff6600); // Orange color for traps
            debugGraphics.strokeRect(x, y, 32, 32); // 32x32 tile size
        }

        const safeFrames = [4, 5, 6, 7, 8];
        let canDamage = true;

        this.physics.add.overlap(s, this.player, (s, p) => {
            if (!canDamage) {
                return;
            }
            const f = s.frame.name;
            console.log('overlap with spikes', f);
            if (!safeFrames.includes(f) && !this.isDead) {
                canDamage = false;
                this.sendGameCommand('HitPlayerCommand', {
                    monsterId: -1,
                    targetClientId: this.myClientId,
                    kind: DAMAGE_KIND_SPIKE
                });
                setTimeout(() => canDamage = true, 1000);
            }
        }, null, this);
        
        const trap = {
            sprite: s,
            debugGraphics: debugGraphics,
            state: 'armed',
            x: x,
            y: y
        };
        
        this.traps = this.traps || {};
        this.traps[trapId] = trap;
        
        return trap;
    },

    XPEvent(data) {
        this.level = data.level;
        this.xp = data.xp;
        this.nextLevelXp = data.nextLevelXp;
        this.updateXpBar();
        if (data.gotNewLevel) {
            const lvlUpText = this.add.text(this.scale.width / 2, this.scale.height / 2, 'LEVEL UP!', { font: '24px Arial', fill: '#00ff00' })
                .setOrigin(0.5, 0.5)
                .setScrollFactor(0, 0)
                .setDepth(DEPTH_UI);
            this.time.delayedCall(2000, () => {
                lvlUpText.destroy();
            });
        }
    },

    HealEvent(data) {
        if (data.clientId !== this.myClientId) return;
        this.player.hp = data.hp;
        if (this.player.hpText) {
            this.player.hpText.setText(data.hp + '/' + data.maxHp);
        }
        if (data.amount > 0) {
            this.showHealText(this.player.x, this.player.y, data.amount);
        }
    },

    InventoryUpdateEvent(data) {
        if (data.clientId !== this.myClientId) return;
        const now = Date.now();
        this.inventory = data.inventory.map(item => {
            if (item.cooldownMs > 0) {
                return Object.assign({}, item, { cooldownEndTime: now + item.cooldownMs });
            }
            return item;
        });
        this.updateItemSelector();
    },

    FootprintsEvent(data) {
        for (const point of data.points) {
            const color = parseInt((point.color || '#ffffff').replace('#', ''), 16);
            const g = this.add.graphics()
                .fillStyle(color, 0.7)
                .fillCircle(point.x, point.y, 5)
                .setDepth(DEPTH_WALLS + 1)
                .setMask(this.mask);
            this.footprintGraphics.push(g);
        }
    },

    FootprintsExpiredEvent(_data) {
        for (const g of this.footprintGraphics) {
            g.destroy();
        }
        this.footprintGraphics = [];
    },

    ProtectionActiveEvent(_data) {
        this._showStatusText('Shield!', '#88aaff');
    },

    ProtectionExpiredEvent(_data) {
        this._showStatusText('Shield expired', '#888888');
    },

    CloakActiveEvent(_data) {
        this._showStatusText('Invisible!', '#aaffee');
    },

    CloakExpiredEvent(_data) {
        this._showStatusText('Visible again', '#888888');
    },

    JellySplitEvent(data) {
        const jelly = this.monsters[data.monsterID];
        if (jelly instanceof Jelly) {
            jelly.triggerSplit(() => {
                delete this.monsters[data.monsterID];
            });
        }
    },

    JellyHitSlowEvent(data) {
        this.player.jellyHitSlowUntil = Date.now() + data.duration;
    },

    _showStatusText(message, color) {
        const x = this.player ? this.player.x : 0;
        const y = this.player ? this.player.y - 20 : 0;
        const text = this.add.text(x, y, message, {
            font: '16px Arial',
            fill: color || '#ffffff',
            stroke: '#000000',
            strokeThickness: 3
        })
        .setDepth(DEPTH_UI)
        .setOrigin(0.5, 0.5);

        this.tweens.add({
            targets: text,
            y: y - 50,
            alpha: 0,
            duration: 1200,
            ease: 'Cubic.easeOut',
            onComplete: () => {
                text.destroy();
            }
        });
    },

    showHealText(x, y, amount) {
        const healText = this.add.text(x, y - 10, `+${amount}`, {
            font: '16px Arial',
            fill: '#00ff00',
            stroke: '#000000',
            strokeThickness: 3
        })
        .setDepth(DEPTH_UI)
        .setOrigin(0.5, 0.5);

        this.tweens.add({
            targets: healText,
            y: y - 50,
            alpha: 0,
            duration: 800,
            ease: 'Cubic.easeOut',
            onComplete: () => {
                healText.destroy();
            }
        });
    }
}
