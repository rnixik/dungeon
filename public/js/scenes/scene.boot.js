var sceneConfigBoot = {
    key: 'boot',
    preload: function() {
        this.load.image('main', 'assets/splash.jpg');
        this.load.image('preloaderBar', 'assets/loader_bar.png');
    },
    create: function() {
        this.scene.switch('Preloader');
    }
};
