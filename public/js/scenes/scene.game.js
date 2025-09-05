class Game extends Phaser.Scene
{
    player;
    cursors;
    rt;

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

        // https://phaser.io/examples/v3.85.0/tilemap/collision/view/tilemap-spotlight
        this.rt = this.add.renderTexture(0, 0, this.scale.width, this.scale.height);
        //  Make sure it doesn't scroll with the camera
        this.rt.setOrigin(0, 0);
        this.rt.setScrollFactor(0, 0);
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

        //  Draw the spotlight on the player
        const cam = this.cameras.main;

        //  Clear the RenderTexture
        this.rt.clear();

        //  Fill it in black
        this.rt.fill(0x000000);

        //  Erase the 'mask' texture from it based on the player position
        //  We - 107, because the mask image is 213px wide, so this puts it on the middle of the player
        //  We then minus the scrollX/Y values, because the RenderTexture is pinned to the screen and doesn't scroll
        this.rt.erase('mask', (this.player.x - 107) - cam.scrollX, (this.player.y - 107) - cam.scrollY);
    }
}

var sceneConfigGame = new Game();
