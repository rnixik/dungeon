class Bullet extends Phaser.Physics.Arcade.Sprite
{
    fire (clientId, monsterId, x, y, velocityVector, animationKey, rotationOffset, distance, useRotation, bodyOptions)
    {
        this.clientId = clientId;
        this.monsterId = monsterId;

        this.enableBody();
        this.body.reset(x, y);

        if (bodyOptions) {
            if (bodyOptions.size) {
                this.body.setSize(bodyOptions.size.x, bodyOptions.size.y);
            }
            if (bodyOptions.offset) {
                this.body.setOffset(bodyOptions.offset.x, bodyOptions.offset.y);
            }
        }

        this.setActive(true);
        this.setVisible(true);

        if (useRotation) {
            const rotationAngle = Math.atan2(velocityVector.y, velocityVector.x);
            this.setRotation(rotationAngle + rotationOffset);
            if (rotationAngle > 90 && rotationAngle < 270) {
                this.setFlipY(true);
            }
        }

        this.setVelocity(velocityVector.x, velocityVector.y);
        if (animationKey) {
            this.anims.play(animationKey, true);
        }

        if (distance) {
            const travelTime = (distance / velocityVector.length()) * 1000;
            this.scene.time.delayedCall(travelTime, () => {
                this.onTravelEnd();
            }, [], this);
        }
    }

    onTravelEnd()
    {
        this.setActive(false);
        this.setVisible(false);
        this.disableBody();
    }
}

class Fireball extends Bullet
{
    hasActiveExplosion = false;

    onTravelEnd() {
        if (this.hasActiveExplosion) {
            return;
        }
        this.hasActiveExplosion = true;

        this.setVisible(false);
        this.disableBody();

        // Colliders registered below are removed when the explosion ends so they
        // don't accumulate in the physics world for the rest of the match.
        const explosionColliders = [];

        const explosion = this.scene.add.sprite(this.x, this.y, 'explosion')
            .setDepth(DEPTH_PROJECTILES)
            .setScale(1.5)
            .setMask(this.scene.mask);
        explosion.anims.play('explosion', true);
        explosion.on('animationcomplete', () => {
            for (const c of explosionColliders) {
                this.scene.physics.world.removeCollider(c);
            }
            explosion.destroy();
            this.setActive(false);
            this.hasActiveExplosion = false;
        });

        if (this.clientId !== this.scene.myClientId) {
            return;
        }

        this.scene.physics.add.existing(explosion);

        // attack my player
        let canDamagePlayer = true;
        explosionColliders.push(this.scene.physics.add.overlap(explosion, this.scene.player, (s, p) => {
            if (!canDamagePlayer) {
                return;
            }
            canDamagePlayer = false;
            this.scene.sendGameCommand('HitPlayerCommand', {
                originClientId: this.scene.myClientId,
                monsterId: -1,
                targetClientId: this.scene.myClientId,
                kind: DAMAGE_KIND_EXPLOSION
            });
            setTimeout(() => canDamagePlayer = false, 1000);
        }, null, this));

        // attack other players
        const playersArray = [];
        for (const id in this.scene.players) {
            playersArray.push(this.scene.players[id]);
        }
        let cannotDamage = {};
        explosionColliders.push(this.scene.physics.add.overlap(explosion, playersArray, (s, p) => {
            if (cannotDamage[p.id]) {
                return;
            }
            cannotDamage[p.id] = true;
            this.scene.sendGameCommand('HitPlayerCommand', {
                originClientId: this.scene.myClientId,
                monsterId: -1,
                targetClientId: p.id,
                kind: DAMAGE_KIND_EXPLOSION
            });
            setTimeout(() => cannotDamage[p.id] = false, 300);
        }, null, this));

        // attack monsters
        const monstersArray = [];
        for (const id in this.scene.monsters) {
            monstersArray.push(this.scene.monsters[id]);
        }

        let cannotDamageMonsters = {};
        explosionColliders.push(this.scene.physics.add.overlap(explosion, monstersArray, (s, m) => {
            if (cannotDamageMonsters[m.id]) {
                return;
            }
            cannotDamageMonsters[m.id] = true;
            this.scene.sendGameCommand('HitMonsterCommand', {
                originClientId: this.clientId,
                monsterId: m.id,
                kind: 'explosion'
            });
            setTimeout(() => cannotDamage[m.id] = false, 1000);
        }, null, this));
    }
}

class Bullets extends Phaser.Physics.Arcade.Group
{
    kind;
    sprites;
    onBulletHitPlayer;
    onBulletHitMonster;
    gameScene;
    animationKey;
    rotationOffset = 0;
    useRotation = true;
    bodyOptions = {};

    constructor (key, animationKey, createConfigOverride, scene, layerWalls, onBulletHitPlayer, onBulletHitMonster)
    {
        super(scene.physics.world, scene);

        const config = Object.assign({
            x: -100,
            y: -100,
            frameQuantity: 10,
            key: key,
            active: false,
            visible: false,
            setDepth: {value: DEPTH_PROJECTILES, step: 0},
            classType: Bullet
        }, createConfigOverride);

        if (createConfigOverride && createConfigOverride.rotationOffset) {
            this.rotationOffset = createConfigOverride.rotationOffset;
        }
        if (createConfigOverride && createConfigOverride.useRotation !== undefined) {
            this.useRotation = createConfigOverride.useRotation;
        }
        if (createConfigOverride && createConfigOverride.bodyOptions) {
            this.bodyOptions = createConfigOverride.bodyOptions;
        }

        this.sprites = this.createMultiple(config);

        scene.physics.add.collider(this.sprites, layerWalls, this.bulletHitWall, null, this);

        this.kind = key;
        this.animationKey = animationKey;
        this.gameScene = scene;
        this.onBulletHitPlayer = onBulletHitPlayer;
        this.onBulletHitMonster = onBulletHitMonster;
    }

    addPlayer (player)
    {
        this.scene.physics.add.overlap(this.sprites, player, this.bulletHitPlayer, null, this);
    }

    addMonster (monster)
    {
        return this.scene.physics.add.overlap(this.sprites, monster, this.bulletHitMonster, null, this);
    }

    addObject (object)
    {
        this.scene.physics.add.overlap(this.sprites, object, this.bulletHitWall, null, this);
    }

    bulletHitWall (bullet, wall)
    {
        if (DEBUG) console.log('hit wall', this.kind);
        this.hideBullet(bullet);
    }

    bulletHitPlayer (bullet, player)
    {
        if (bullet.clientId === player.id)
        {
            // don't hit yourself
            return;
        }
        this.hideBullet(bullet);
        this.onBulletHitPlayer.apply(this.gameScene, [bullet, player, this.kind]);
    }

    bulletHitMonster (bullet, monster)
    {
        if (bullet.monsterId === monster.id) {
            // don't hit yourself
            return;
        }
        this.hideBullet(bullet);
        this.onBulletHitMonster.apply(this.gameScene, [bullet, monster, this.kind]);
    }

    hideBullet (bullet)
    {
        bullet.onTravelEnd();
    }

    fireBullet (clientId, monsterId, x, y, vector, distance)
    {
        const bullet = this.getFirstDead(true);
        if (bullet) {
            bullet.fire(clientId, monsterId, x, y, vector, this.animationKey, this.rotationOffset, distance, this.useRotation, this.bodyOptions);
        }

        return bullet;
    }

    shootToDirection4x(clientId, monsterId, x, y, direction4x, velocity, distance) {
        let vector = new Phaser.Math.Vector2(1, 0);
        switch (direction4x) {
            case 'left': vector = new Phaser.Math.Vector2(-1, 0); break;
            case 'right': vector = new Phaser.Math.Vector2(1, 0); break;
            case 'up': vector = new Phaser.Math.Vector2(0, -1); break;
            case 'down': vector = new Phaser.Math.Vector2(0, 1); break;
        }
        vector = vector.normalize().scale(velocity);

        return this.fireBullet(clientId, monsterId, x, y, vector, distance);
    }

    shootToVector(clientId, monsterId, x, y, vector, velocity) {
        vector = vector.normalize().scale(velocity);

        return this.fireBullet(clientId, monsterId, x, y, vector);
    }

    shootToPoint(clientId, monsterId, x, y, destX, destY, velocity) {
        let vector = new Phaser.Math.Vector2(destX - x, destY - y);
        vector = vector.normalize().scale(velocity);

        return this.fireBullet(clientId, monsterId, x, y, vector);
    }
}

class FireballsGroup extends Bullets
{
    constructor (scene, layerWalls, onBulletHitPlayer, onBulletHitMonster)
    {
        super('fireball',
            'fireball-loop',
            {frameQuantity: 20, setScale: {x: 0.7, y: 0.7}, classType: Fireball},
            scene,
            layerWalls,
            onBulletHitPlayer,
            onBulletHitMonster);
    }
}

class FireboltsGroup extends Bullets
{
    constructor (scene, layerWalls, onBulletHitPlayer, onBulletHitMonster)
    {
        super('bullet', null, {frameQuantity: 16}, scene, layerWalls, onBulletHitPlayer, onBulletHitMonster);
    }
}

class FirespotsGroup extends Bullets
{
    constructor (scene, layerWalls, onBulletHitPlayer, onBulletHitMonster)
    {
        super('firespot', 'firespot', {
            frameQuantity: 16,
            setScale: {x: 3, y: 3},
            rotationOffset: - Math.PI / 2,
            useRotation: false,
            bodyOptions: {
                size: {x: 15, y: 20},
                offset: {x: 25, y: 35}
            }
        }, scene, layerWalls, onBulletHitPlayer, onBulletHitMonster);
    }
}

class ArrowsGroup extends Bullets
{
    constructor (scene, layerWalls, onBulletHitPlayer, onBulletHitMonster)
    {
        super(
            'arrow',
            null,
            {frameQuantity: 30, setScale: {x: 2, y: 2}, rotationOffset: -Math.PI / 2},
            scene,
            layerWalls,
            onBulletHitPlayer,
            onBulletHitMonster
        );
    }
}

class AllProjectilesGroup
{
    fireballs;
    firebolts;
    firespots;
    arrows;

    constructor (scene, layerWalls, onBulletHitPlayer, onBulletHitMonster)
    {
        this.fireballs = new FireballsGroup(scene, layerWalls, onBulletHitPlayer, onBulletHitMonster);
        this.firebolts = new FireboltsGroup(scene, layerWalls, onBulletHitPlayer, onBulletHitMonster);
        this.firespots = new FirespotsGroup(scene, layerWalls, onBulletHitPlayer, onBulletHitMonster);
        this.arrows = new ArrowsGroup(scene, layerWalls, onBulletHitPlayer, onBulletHitMonster);
    }

    addPlayer(player) {
        this.fireballs.addPlayer(player);
        this.firebolts.addPlayer(player);
        this.firespots.addPlayer(player);
        this.arrows.addPlayer(player);
    }

    addMonster(monster) {
        // Keep references so they can be removed from the physics world when the
        // monster dies — otherwise stale colliders accumulate for the whole match.
        monster._projectileColliders = [
            this.fireballs.addMonster(monster),
            this.arrows.addMonster(monster),
        ];
    }

    addObject(object) {
        this.fireballs.addObject(object);
        this.arrows.addObject(object);
    }

    getAllIlluminatedSprites() {
        const children = this.fireballs.getChildren();
        return children.filter(b => b.active);
    }

    castPlayerFireball(clientId, x, y, direction4x, velocity, distance)
    {
        return this.fireballs.shootToDirection4x(clientId, null, x, y, direction4x, velocity, distance);
    }

    castMonsterFireball(monsterId, x, y, destX, destY, velocity)
    {
        return this.fireballs.shootToPoint(null, monsterId, x, y, destX, destY, velocity);
    }

    castMonsterFirebolt(monsterId, x, y, destX, destY, velocity)
    {
        return this.firebolts.shootToPoint(null, monsterId, x, y, destX, destY, velocity);
    }

    castMonsterFireboltToVector(monsterId, x, y, vector, velocity)
    {
        return this.firebolts.shootToVector(null, monsterId, x, y, vector, velocity);
    }

    castMonsterFirespotVector(monsterId, x, y, vector, velocity)
    {
        return this.firespots.shootToVector(null, monsterId, x, y, vector, velocity);
    }

    shootMonsterArrow(monsterId, x, y, destX, destY, velocity)
    {
        return this.arrows.shootToPoint(null, monsterId, x, y, destX, destY, velocity);
    }

    shootPlayerArrow(clientId, x, y, destX, destY, velocity)
    {
        return this.arrows.shootToPoint(clientId, null, x, y, destX, destY, velocity);
    }
}

class DemonLightningGroup
{
    constructor (monsterId, x, y, targetX, targetY, scene)
    {
        const warmupMs = 800; // warning window before first bolt

        // Demon glows red for the entire warmup period
        const demonSprite = scene.monsters && scene.monsters[monsterId];
        if (demonSprite) {
            demonSprite.setTint(0xff0000);
            scene.time.delayedCall(warmupMs, () => {
                if (demonSprite && !demonSprite.isCorpse) demonSprite.clearTint();
            });
        }

        // Glowing warning spot at target — pixel-art concentric squares, pulses during warmup
        const spot = scene.add.graphics();
        spot.fillStyle(0x440000, 0.30); spot.fillRect(-22, -22, 44, 44); // outer soft glow
        spot.fillStyle(0x880000, 1.00); spot.fillRect(-14, -14, 28, 28); // ring 1
        spot.fillStyle(0xbb1111, 1.00); spot.fillRect(-10, -10, 20, 20); // ring 2
        spot.fillStyle(0xff3333, 1.00); spot.fillRect( -7,  -7, 14, 14); // ring 3
        spot.fillStyle(0xff7777, 1.00); spot.fillRect( -4,  -4,  8,  8); // ring 4
        spot.fillStyle(0xffbbbb, 1.00); spot.fillRect( -2,  -2,  4,  4); // ring 5
        spot.fillStyle(0xffffff, 1.00); spot.fillRect( -1,  -1,  2,  2); // center

        const spotCtr = scene.add.container(targetX, targetY, [spot])
            .setDepth(DEPTH_PROJECTILES - 1)
            .setAlpha(0)
            .setMask(scene.mask);

        // Fade in
        scene.tweens.add({ targets: spotCtr, alpha: 1, duration: 200, ease: 'Cubic.easeOut' });

        // Pulse
        const pulseTween = scene.tweens.add({
            targets: spotCtr, scaleX: 1.35, scaleY: 1.35,
            duration: 260, yoyo: true, repeat: -1, ease: 'Sine.easeInOut'
        });

        // Burst and vanish just as the first bolt fires
        scene.time.delayedCall(warmupMs - 100, () => {
            pulseTween.stop();
            scene.tweens.add({
                targets: spotCtr, scaleX: 2.6, scaleY: 2.6, alpha: 0,
                duration: 160, ease: 'Cubic.easeOut',
                onComplete: () => spotCtr.destroy()
            });
        });

        let damagedInThisAttack = false;

        for (let i = 0; i < 5; i++) {
            scene.time.delayedCall(warmupMs + i * 350, () => {
                const tx = targetX + (Math.random() - 0.5) * 14;
                const ty = targetY + (Math.random() - 0.5) * 14;
                this._spawnBolt(x, y, tx, ty, scene);

                scene.cameras.main?.shake(i === 0 ? 90 : 45, i === 0 ? 0.009 : 0.004);

                // Damage: one hit maximum per 5-bolt sequence
                if (!damagedInThisAttack && !scene.isDead && scene.player) {
                    const dist = this._distToSegment(
                        scene.player.x, scene.player.y,
                        x, y, targetX, targetY
                    );
                    if (dist < 25) {
                        damagedInThisAttack = true;
                        scene.sendGameCommand('HitPlayerCommand', {
                            monsterId: monsterId,
                            targetClientId: scene.myClientId,
                            kind: DAMAGE_KIND_LIGHTNING,
                        });
                    }
                }
            });
        }
    }

    _spawnBolt (x1, y1, x2, y2, scene)
    {
        const points = this._generateLightningPoints(x1, y1, x2, y2);
        const graphics = scene.add.graphics().setDepth(DEPTH_PROJECTILES).setMask(scene.mask);

        // Layered glow: outer dark → mid → bright core → white-hot centre
        const layers = [
            { width: 9,  color: 0x550000, alpha: 0.22 },
            { width: 5,  color: 0xaa0000, alpha: 0.55 },
            { width: 2,  color: 0xff3333, alpha: 0.95 },
            { width: 1,  color: 0xffaaaa, alpha: 1.0  },
        ];
        for (const { width, color, alpha } of layers) {
            graphics.lineStyle(width, color, alpha);
            graphics.beginPath();
            graphics.moveTo(points[0].x, points[0].y);
            for (let i = 1; i < points.length; i++) graphics.lineTo(points[i].x, points[i].y);
            graphics.strokePath();
        }

        // Branches
        const dx = x2 - x1, dy = y2 - y1;
        const len = Math.hypot(dx, dy) || 1;
        const numBranches = Phaser.Math.Between(1, 2);
        for (let b = 0; b < numBranches; b++) {
            const bpi = Phaser.Math.Between(2, points.length - 2);
            const branchLen = len * (0.1 + Math.random() * 0.15);
            const angle = Math.atan2(dy, dx) + (Math.random() - 0.5) * Math.PI * 0.6;
            graphics.lineStyle(1, 0xff4444, 0.7);
            graphics.beginPath();
            graphics.moveTo(points[bpi].x, points[bpi].y);
            graphics.lineTo(points[bpi].x + Math.cos(angle) * branchLen, points[bpi].y + Math.sin(angle) * branchLen);
            graphics.strokePath();
        }

        scene.tweens.add({
            targets: graphics,
            alpha: 0,
            duration: 220,
            delay: 40,
            ease: 'Cubic.easeIn',
            onComplete: () => graphics.destroy()
        });
    }

    _generateLightningPoints (x1, y1, x2, y2, segments)
    {
        segments = segments || 10;
        const dx = x2 - x1, dy = y2 - y1;
        const len = Math.hypot(dx, dy) || 1;
        const perpX = -dy / len, perpY = dx / len;
        const points = [{ x: x1, y: y1 }];
        for (let i = 1; i < segments; i++) {
            const t = i / segments;
            const jitter = (Math.random() - 0.5) * len * 0.22;
            points.push({ x: x1 + dx * t + perpX * jitter, y: y1 + dy * t + perpY * jitter });
        }
        points.push({ x: x2, y: y2 });
        return points;
    }

    _distToSegment (px, py, x1, y1, x2, y2)
    {
        const dx = x2 - x1, dy = y2 - y1;
        const lenSq = dx * dx + dy * dy;
        if (lenSq === 0) return Math.hypot(px - x1, py - y1);
        const t = Math.max(0, Math.min(1, ((px - x1) * dx + (py - y1) * dy) / lenSq));
        return Math.hypot(px - (x1 + t * dx), py - (y1 + t * dy));
    }
}