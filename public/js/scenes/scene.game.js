class Game extends Phaser.Scene
{
    controls;

    constructor ()
    {
        super({ key: 'Game' });
    }

    create ()
    {
        const map = this.make.tilemap({ key: 'map' });

        const tiles = map.addTilesetImage('tiles_atlas', 'tiles');

        map.createLayer(0, tiles, 0, 0); // floor
        map.createLayer(1, tiles, 0, 0); // walls

        this.cameras.main.setBounds(0, 0, map.widthInPixels, map.heightInPixels);

        const cursors = this.input.keyboard.createCursorKeys();

        const controlConfig = {
            camera: this.cameras.main,
            left: cursors.left,
            right: cursors.right,
            up: cursors.up,
            down: cursors.down,
            speed: 0.5
        };

        this.controls = new Phaser.Cameras.Controls.FixedKeyControl(controlConfig);
    }

    update (time, delta)
    {
        this.controls.update(delta);
    }
}

var sceneConfigGame = new Game();
