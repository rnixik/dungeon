class Game extends Phaser.Scene
{
    player;
    cursors;

    constructor ()
    {
        super({ key: 'Game' });
    }

    create ()
    {
        const map = this.make.tilemap({ key: 'map' });

        const tiles = map.addTilesetImage('tiles_atlas', 'tiles');

        map.createLayer(0, tiles, 0, 0); // floor
        let layerWalls = map.createLayer(1, tiles, 0, 0); // walls
        // all tiles can collide, we just use collider for layer
        map.setCollisionBetween(0, 5);

        this.player = this.physics.add.sprite(50, 140, 'player', 1);
        this.player.setScale(1.5);

        this.physics.add.collider(this.player, layerWalls);

        this.cameras.main.setBounds(0, 0, map.widthInPixels, map.heightInPixels);
        this.cameras.main.startFollow(this.player);

        this.cursors = this.input.keyboard.createCursorKeys();
    }

    update (time, delta)
    {
        this.player.body.setVelocity(0);

        // Horizontal movement
        if (this.cursors.left.isDown)
        {
            this.player.body.setVelocityX(-100);
        }
        else if (this.cursors.right.isDown)
        {
            this.player.body.setVelocityX(100);
        }

        // Vertical movement
        if (this.cursors.up.isDown)
        {
            this.player.body.setVelocityY(-100);
        }
        else if (this.cursors.down.isDown)
        {
            this.player.body.setVelocityY(100);
        }

        // Update the animation last and give left/right animations precedence over up/down animations
        if (this.cursors.left.isDown)
        {
            this.player.anims.play('left', true);
        }
        else if (this.cursors.right.isDown)
        {
            this.player.anims.play('right', true);
        }
        else if (this.cursors.up.isDown)
        {
            this.player.anims.play('up', true);
        }
        else if (this.cursors.down.isDown)
        {
            this.player.anims.play('down', true);
        }
        else
        {
            this.player.anims.stop();
        }
    }
}

var sceneConfigGame = new Game();
