class Bullet extends Phaser.Physics.Arcade.Sprite
{
    constructor (scene, x, y)
    {
        super(scene, x, y, 'bullet');
    }

    fire (x, y, direction)
    {
        this.body.reset(x, y);

        this.setActive(true);
        this.setVisible(true);

        const vel = 500;

        let velX = 0;
        let velY = 0;
        if (direction === 'left')
        {
            velX = -vel;
        }
        else if (direction === 'right')
        {
            velX = vel;
        }
        else if (direction === 'up')
        {
            velY = -vel;
        }
        else if (direction === 'down')
        {
            velY = vel;
        }
        this.setVelocity(velX, velY);
    }

    preUpdate (time, delta)
    {
        super.preUpdate(time, delta);

        if (this.y <= -32 || this.y >= 2000 || this.x <= -32 || this.x >= 2000)
        {
            this.setActive(false);
            this.setVisible(false);
        }
    }
}

class Bullets extends Phaser.Physics.Arcade.Group
{
    constructor (scene, layerWalls)
    {
        super(scene.physics.world, scene);

        const bullets = this.createMultiple({
            frameQuantity: 50,
            key: 'bullet',
            active: false,
            visible: false,
            classType: Bullet
        });

        scene.physics.add.collider(bullets, layerWalls, this.bulletHitWall, null, this);
    }

    bulletHitWall (bullet, wall)
    {
        console.log('hit');
        bullet.setActive(false);
        bullet.setVisible(false);
    }

    fireBullet (x, y, direction)
    {
        const bullet = this.getFirstDead(false);

        if (bullet)
        {
            bullet.fire(x, y, direction);
        }
    }
}