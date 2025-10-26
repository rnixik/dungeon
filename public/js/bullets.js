class Bullet extends Phaser.Physics.Arcade.Sprite
{
    fire (clientId, monsterId, x, y, velocityVector, animationKey, rotationOffset)
    {
        this.clientId = clientId;
        this.monsterId = monsterId;

        this.enableBody();
        this.body.reset(x, y);

        this.setActive(true);
        this.setVisible(true);

        const rotationAngle = Math.atan2(velocityVector.y, velocityVector.x);
        this.setRotation(rotationAngle + rotationOffset);
        if (rotationAngle > 90 && rotationAngle < 270) {
            this.setFlipY(true);
        }

        this.setVelocity(velocityVector.x, velocityVector.y);
        if (animationKey) {
            this.anims.play(animationKey, true);
        }
    }
}

class Bullets extends Phaser.Physics.Arcade.Group
{
    sprites;
    onBulletHitPlayer;
    onBulletHitMonster;
    gameScene;
    animationKey;
    rotationOffset = 0;

    constructor (key, animationKey, createConfigOverride, scene, layerWalls, onBulletHitPlayer, onBulletHitMonster)
    {
        super(scene.physics.world, scene);

        const config = Object.assign({
            frameQuantity: 100,
            key: key,
            active: false,
            visible: false,
            setDepth: {value: DEPTH_PROJECTILES, step: 0},
            classType: Bullet
        }, createConfigOverride);

        if (createConfigOverride && createConfigOverride.rotationOffset) {
            this.rotationOffset = createConfigOverride.rotationOffset;
        }

        this.sprites = this.createMultiple(config);

        scene.physics.add.collider(this.sprites, layerWalls, this.bulletHitWall, null, this);

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
        this.scene.physics.add.overlap(this.sprites, monster, this.bulletHitMonster, null, this);
    }

    addObject (object)
    {
        this.scene.physics.add.overlap(this.sprites, object, this.bulletHitWall, null, this);
    }

    bulletHitWall (bullet, wall)
    {
        console.log('hit wall');
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
        this.onBulletHitPlayer.apply(this.gameScene, [bullet, player]);
    }

    bulletHitMonster (bullet, monster)
    {
        if (bullet.monsterId === monster.id) {
            // don't hit yourself
            return;
        }
        this.hideBullet(bullet);
        this.onBulletHitMonster.apply(this.gameScene, [bullet, monster]);
    }

    hideBullet (bullet)
    {
        bullet.setActive(false);
        bullet.setVisible(false);
        bullet.disableBody();
    }

    fireBullet (clientId, monsterId, x, y, vector)
    {
        const bullet = this.getFirstDead(true);
        if (bullet) {
            bullet.fire(clientId, monsterId, x, y, vector, this.animationKey, this.rotationOffset);
        }

        return bullet;
    }

    shootToDirection4x(clientId, monsterId, x, y, direction4x, velocity) {
        let vector = new Phaser.Math.Vector2(1, 0);
        switch (direction4x) {
            case 'left': vector = new Phaser.Math.Vector2(-1, 0); break;
            case 'right': vector = new Phaser.Math.Vector2(1, 0); break;
            case 'up': vector = new Phaser.Math.Vector2(0, -1); break;
            case 'down': vector = new Phaser.Math.Vector2(0, 1); break;
        }
        vector = vector.normalize().scale(velocity);

        return this.fireBullet(clientId, monsterId, x, y, vector);
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
            {setScale: {x: 0.7, y: 0.7}},
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
        super('bullet', null, {}, scene, layerWalls, onBulletHitPlayer, onBulletHitMonster);
    }
}

class ArrowsGroup extends Bullets
{
    constructor (scene, layerWalls, onBulletHitPlayer, onBulletHitMonster)
    {
        super(
            'arrow',
            null,
            {setScale: {x: 2, y: 2}, rotationOffset: -Math.PI / 2},
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
    arrows;

    constructor (scene, layerWalls, onBulletHitPlayer, onBulletHitMonster)
    {
        this.fireballs = new FireballsGroup(scene, layerWalls, onBulletHitPlayer, onBulletHitMonster);
        this.firebolts = new FireboltsGroup(scene, layerWalls, onBulletHitPlayer, onBulletHitMonster);
        this.arrows = new ArrowsGroup(scene, layerWalls, onBulletHitPlayer, onBulletHitMonster);
    }

    addPlayer(player) {
        this.fireballs.addPlayer(player);
        this.firebolts.addPlayer(player);
        this.arrows.addPlayer(player);
    }

    addMonster(monster) {
        this.fireballs.addMonster(monster);
        this.firebolts.addMonster(monster);
        this.arrows.addMonster(monster);
    }

    addObject(object) {
        this.fireballs.addObject(object);
        this.firebolts.addObject(object);
        this.arrows.addObject(object);
    }

    getAllIlluminatedSprites() {
        const children = this.fireballs.getChildren();
        return children.filter(b => b.active);
    }

    castPlayerFireball(clientId, x, y, direction4x, velocity)
    {
        return this.fireballs.shootToDirection4x(clientId, null, x, y, direction4x, velocity);
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

    shootMonsterArrow(monsterId, x, y, destX, destY, velocity)
    {
        return this.arrows.shootToPoint(null, monsterId, x, y, destX, destY, velocity);
    }
}

class LightingGroup
{
    onBulletHitPlayer;
    gameScene;

    constructor (monsterId, x, y, scene)
    {
        const lightningLength = 280;
        const lightningSpeed = 600;

        const lightningL = scene.physics.add.sprite(x - lightningLength, y, 'lightning')
            .setOrigin(0, 0.5)
            .setFlipX(true)
            .setAlpha(0.7)
            .setDepth(DEPTH_PROJECTILES)
            .setMask(scene.mask)
            .setVelocity(-lightningSpeed, 0);
        lightningL.monsterId = monsterId;
        lightningL.isHorizontal = true;
        scene.physics.add.overlap(lightningL, scene.player, this.bulletHitPlayer, null, this);
        scene.physics.add.collider(lightningL, scene.layerWalls, this.bulletHitWall, null, this);

        const lightningD = scene.physics.add.sprite(x, y + lightningLength, 'lightning_v')
            .setOrigin(0.5, 1)
            .setAlpha(0.7)
            .setDepth(DEPTH_PROJECTILES)
            .setMask(scene.mask)
            .setVelocity(0, lightningSpeed);
        lightningD.monsterId = monsterId;
        lightningD.isVertical = true;
        scene.physics.add.overlap(lightningD, scene.player, this.bulletHitPlayer, null, this);
        scene.physics.add.collider(lightningD, scene.layerWalls, this.bulletHitWall, null, this);

        const lightningR = scene.physics.add.sprite(x + lightningLength, y, 'lightning')
            .setOrigin(1, 0.5)
            .setAlpha(0.7)
            .setDepth(DEPTH_PROJECTILES)
            .setMask(scene.mask)
            .setVelocity(lightningSpeed, 0);
        lightningR.monsterId = monsterId;
        lightningR.isHorizontal = true;
        scene.physics.add.overlap(lightningR, scene.player, this.bulletHitPlayer, null, this);
        scene.physics.add.collider(lightningR, scene.layerWalls, this.bulletHitWall, null, this);

        const lightningU = scene.physics.add.sprite(x, y - lightningLength, 'lightning_v')
            .setOrigin(0.5, 0)
            .setFlipY(true)
            .setAlpha(0.7)
            .setDepth(DEPTH_PROJECTILES)
            .setMask(scene.mask)
            .setVelocity(0, -lightningSpeed);
        lightningU.monsterId = monsterId;
        lightningU.isVertical = true;
        scene.physics.add.overlap(lightningU, scene.player, this.bulletHitPlayer, null, this);
        scene.physics.add.collider(lightningU, scene.layerWalls, this.bulletHitWall, null, this);

        // destroy after some time
        scene.time.delayedCall(10000, () => {
            lightningL.destroy();
            lightningD.destroy();
            lightningR.destroy();
            lightningU.destroy();
        }, [], this);

        this.gameScene = scene;
        this.onBulletHitPlayer = scene.onBulletHitPlayer;
    }

    bulletHitWall (bullet, wall)
    {
        let props = {scaleX: 0};
        if (bullet.isVertical) {
            props = {scaleY: 0};
        }
        this.gameScene.tweens.add({
            targets: bullet,
            props: props,
            duration: 200,
            ease: 'Linear',
            onComplete: () => {
                bullet.destroy();
            }
        });
    }

    bulletHitPlayer (bullet, player)
    {
        if (bullet.alreadyHitPlayer) {
            return;
        }

        bullet.alreadyHitPlayer = true;
        this.onBulletHitPlayer.apply(this.gameScene, [bullet, player]);
    }
}