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
        this.load.tilemapTiledJSON('map', 'assets/dungeon1.tmj');

        this.load.image('archer', 'assets/archer.png');

    },
    create: function() {
        // a place to create animations if needed in the future
        // this.scene.switch('MainMenu');
        this.scene.switch('Game');
    }
};
