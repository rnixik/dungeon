class Bullet extends Phaser.Physics.Arcade.Sprite
{
    fire (clientId, x, y, velocityVector, animationKey)
    {
        this.clientId = clientId;

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
        this.anims.play(animationKey, true);
    }
}

class Bullets extends Phaser.Physics.Arcade.Group
{
    sprites;
    onBulletHitPlayer;
    onBulletHitMonster;
    gameScene;

    constructor (key, scene, layerWalls, onBulletHitPlayer, onBulletHitMonster)
    {
        super(scene.physics.world, scene);

        this.sprites = this.createMultiple({
            frameQuantity: 100,
            key: key,
            active: false,
            visible: false,
            setDepth: {value: DEPTH_PROJECTILES, step: 0},
            classType: Bullet
        });

        scene.physics.add.collider(this.sprites, layerWalls, this.bulletHitWall, null, this);

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
        this.hideBullet(bullet);
        this.onBulletHitMonster.apply(this.gameScene, [bullet, monster]);
    }

    hideBullet (bullet)
    {
        bullet.setActive(false);
        bullet.setVisible(false);
        bullet.disableBody();
    }

    fireBullet (clientId, x, y, vector, animationKey)
    {
        const bullet = this.getFirstDead(true);
        if (bullet) {
            bullet.fire(clientId, x, y, vector, animationKey);
        }

        return bullet;
    }
}

class FireballsGroup extends Bullets
{
    constructor (scene, layerWalls, onBulletHitPlayer, onBulletHitMonster)
    {
        super('fireball', scene, layerWalls, onBulletHitPlayer, onBulletHitMonster);
    }

    fireBullet(clientId, x, y, direction4x, velocity) {
        let vector = new Phaser.Math.Vector2(1, 0);
        switch (direction4x) {
            case 'left': vector = new Phaser.Math.Vector2(-1, 0); break;
            case 'right': vector = new Phaser.Math.Vector2(1, 0); break;
            case 'up': vector = new Phaser.Math.Vector2(0, -1); break;
            case 'down': vector = new Phaser.Math.Vector2(0, 1); break;
        }
        vector = vector.normalize().scale(velocity);

        return super.fireBullet(clientId, x, y, vector, 'fireball-loop');
    }
}

class AllProjectilesGroup
{
    fireballs;

    constructor (scene, layerWalls, onBulletHitPlayer, onBulletHitMonster)
    {
        this.fireballs = new FireballsGroup(scene, layerWalls, onBulletHitPlayer, onBulletHitMonster);
    }

    addPlayer(player) {
        this.fireballs.addPlayer(player);
    }

    addMonster(monster) {
        this.fireballs.addMonster(monster);
    }

    addObject(object) {
        this.fireballs.addObject(object);
    }

    getAllIlluminatedSprites() {
        const children = this.fireballs.getChildren();
        return children.filter(b => b.active);
    }

    castFireball(clientId, x, y, direction4x, velocity)
    {
        return this.fireballs.fireBullet(clientId, x, y, direction4x, velocity);
    }
}