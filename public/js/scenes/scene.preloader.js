var sceneConfigPreloader = {
    key: 'Preloader',
    preload: function() {
        this.cameras.main.backgroundColor = Phaser.Display.Color.HexStringToColor("#16181a");
        this.add.sprite(0, 0, 'main').setOrigin(0,0).setScale(2);

        const preloadBar = this.add.sprite(145, this.cameras.main.height - 165, 'preloaderBar').setOrigin(0,0);

        this.load.on('progress', function (value) {
            preloadBar.setCrop(0, 0, preloadBar.width * value, preloadBar.height);
        });

        this.load.image('tiles', 'assets/atlas.png');
        this.load.tilemapTiledJSON('map', 'assets/dungeon1.tmj?v=2');

        this.load.spritesheet('player', 'assets/spaceman.png', { frameWidth: 16, frameHeight: 16 });
        this.load.image('mask', 'assets/mask1.png?v=2');
        this.load.atlas('controls', 'assets/controls.png', 'assets/controls.json');
    },

    create: function() {
        this.anims.create({
            key: 'left',
            frames: this.anims.generateFrameNumbers('player', { start: 8, end: 9 }),
            frameRate: 10,
            repeat: -1
        });
        this.anims.create({
            key: 'right',
            frames: this.anims.generateFrameNumbers('player', { start: 1, end: 2 }),
            frameRate: 10,
            repeat: -1
        });
        this.anims.create({
            key: 'up',
            frames: this.anims.generateFrameNumbers('player', { start: 11, end: 13 }),
            frameRate: 10,
            repeat: -1
        });
        this.anims.create({
            key: 'down',
            frames: this.anims.generateFrameNumbers('player', { start: 4, end: 6 }),
            frameRate: 10,
            repeat: -1
        });

        // this.scene.switch('MainMenu');
        this.scene.switch('Game');
    }
};
