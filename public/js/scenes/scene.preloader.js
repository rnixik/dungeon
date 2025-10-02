var sceneConfigPreloader = {
    key: 'Preloader',
    preload: function() {
        this.cameras.main.backgroundColor = Phaser.Display.Color.HexStringToColor("#16181a");
        this.add.sprite(0, 0, 'main').setOrigin(0,0).setScale(2);

        const preloadBar = this.add.sprite(145, this.cameras.main.height - 165, 'preloaderBar').setOrigin(0,0);

        this.load.on('progress', function (value) {
            preloadBar.setCrop(0, 0, preloadBar.width * value, preloadBar.height);
        });

        this.load.image('tiles', 'assets/environment.png');
        this.load.tilemapTiledJSON('map', 'assets/dungeon1.tmj?v=2');

        this.load.spritesheet('player', 'assets/MiniRouge\\3 - Heroes\\Hero 01 Mage\\32x32\\Hero01 Mage Idle2x-Sheet.png', { frameWidth: 32, frameHeight: 32 });
        this.load.spritesheet('fireball', 'assets/MiniRouge\\3 - Heroes\\Hero 01 Mage\\32x32\\Fireball Magel-Sheet2x.png', { frameWidth: 32, frameHeight: 32 });
        this.load.image('mask', 'assets/mask1.png?v=2');
        this.load.atlas('controls', 'assets/controls.png', 'assets/controls.json');
        this.load.image('bullet', 'assets/bullet7.png');
        this.load.image('spinner', 'assets/spinner.png');
        this.load.image('archer', 'assets/archer.png');

        this.load.plugin('rexvirtualjoystickplugin', 'js/rexvirtualjoystickplugin.min.js', true);
    },

    create: function() {
        this.anims.create({
            key: 'left',
            frames: this.anims.generateFrameNumbers('player', { start: 0, end: 11 }),
            frameRate: 15,
            repeat: -1
        });
        this.anims.create({
            key: 'right',
            frames: this.anims.generateFrameNumbers('player', { start: 0, end: 11 }),
            frameRate: 15,
            repeat: -1
        });
        this.anims.create({
            key: 'up',
            frames: this.anims.generateFrameNumbers('player', { start: 13, end: 17 }),
            frameRate: 15,
            repeat: -1
        });
        this.anims.create({
            key: 'down',
            frames: this.anims.generateFrameNumbers('player', { start: 24, end: 35 }),
            frameRate: 15,
            repeat: -1
        });
        this.anims.create({
            key: 'idle',
            frames: this.anims.generateFrameNumbers('player', { start: 1, end: 11 }),
            frameRate: 5,
            repeat: -1
        });
        this.anims.create({
            key: 'fireball-loop',
            frames: this.anims.generateFrameNumbers('fireball', { start: 1, end: 12 }),
            frameRate: 15,
            repeat: -1
        })

        this.scene.switch('MainMenu');
    }
};
