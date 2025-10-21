class Bullet extends Phaser.Physics.Arcade.Sprite
{
    fire (clientId, monsterId, x, y, velocityVector, animationKey)
    {
        this.clientId = clientId;
        this.monsterId = monsterId;

        this.enableBody();
        this.body.reset(x, y);

        this.setActive(true);
        this.setVisible(true);

        const rotationAngle = Math.atan2(velocityVector.y, velocityVector.x);
        this.setRotation(rotationAngle);
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

    constructor (key, animationKey, scale, scene, layerWalls, onBulletHitPlayer, onBulletHitMonster)
    {
        super(scene.physics.world, scene);

        this.sprites = this.createMultiple({
            frameQuantity: 100,
            key: key,
            active: false,
            visible: false,
            setDepth: {value: DEPTH_PROJECTILES, step: 0},
            setScale: {x: scale, y: scale},
            classType: Bullet
        });

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
            bullet.fire(clientId, monsterId, x, y, vector, this.animationKey);
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
        super('fireball', 'fireball-loop', 1, scene, layerWalls, onBulletHitPlayer, onBulletHitMonster);
    }
}

class ArrowsGroup extends Bullets
{
    constructor (scene, layerWalls, onBulletHitPlayer, onBulletHitMonster)
    {
        super('arrow', null, 2, scene, layerWalls, onBulletHitPlayer, onBulletHitMonster);
    }
}

class AllProjectilesGroup
{
    fireballs;
    arrows;

    constructor (scene, layerWalls, onBulletHitPlayer, onBulletHitMonster)
    {
        this.fireballs = new FireballsGroup(scene, layerWalls, onBulletHitPlayer, onBulletHitMonster);
        this.arrows = new ArrowsGroup(scene, layerWalls, onBulletHitPlayer, onBulletHitMonster);
    }

    addPlayer(player) {
        this.fireballs.addPlayer(player);
        this.arrows.addPlayer(player);
    }

    addMonster(monster) {
        this.fireballs.addMonster(monster);
        this.arrows.addMonster(monster);
    }

    addObject(object) {
        this.fireballs.addObject(object);
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

    shootMonsterArrow(monsterId, x, y, destX, destY, velocity)
    {
        return this.arrows.shootToPoint(null, monsterId, x, y, destX, destY, velocity);
    }
}