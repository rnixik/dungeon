class Bullet extends Phaser.Physics.Arcade.Sprite
{
    constructor (scene, x, y)
    {
        super(scene, x, y, 'fireball');
    }

    fire (clientId, x, y, direction)
    {
        this.clientId = clientId;

        this.enableBody();
        this.body.reset(x, y);

        this.setActive(true);
        this.setVisible(true);

        const vel = 500;

        let velX = 0;
        let velY = 0;
        if (direction === 'left')
        {
            velX = -vel;
            this.setAngle(0).setFlipX(true);
        }
        else if (direction === 'right')
        {
            velX = vel;
            this.setAngle(0).setFlipX(false);
        }
        else if (direction === 'up')
        {
            velY = -vel;
            this.setAngle(-90).setFlipX(false);
        }
        else if (direction === 'down')
        {
            velY = vel;
            this.setAngle(90).setFlipX(false);
        }
        this.setVelocity(velX, velY);

        this.anims.play('fireball-loop',true);
    }
}

class Bullets extends Phaser.Physics.Arcade.Group
{
    sprites;
    onBulletHitPlayer;
    onBulletHitMonster;
    gameScene;

    constructor (scene, layerWalls, onBulletHitPlayer, onBulletHitMonster)
    {
        super(scene.physics.world, scene);

        this.sprites = this.createMultiple({
            frameQuantity: 100,
            key: 'fireball',
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

    fireBullet (clientId, x, y, direction)
    {
        const bullet = this.getFirstDead(true);

        if (bullet)
        {
            bullet.fire(clientId, x, y, direction);
        }
    }
}