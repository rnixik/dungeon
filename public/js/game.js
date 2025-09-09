(function() {
    const config = {
        type: Phaser.AUTO,
        width: 1600,
        height: 1200,
        pixelArt: true,
        physics: {
            default: 'arcade',
            arcade: {
                debug: false
            }
        },
        scale: {
            mode: Phaser.Scale.RESIZE,
            autoCenter: Phaser.Scale.CENTER_BOTH,
            min: {
                width: 320,
                height: 240
            },
            max: {
                width: 1600,
                height: 1200
            }
        },
        scene: [ sceneConfigBoot, sceneConfigPreloader, sceneConfigGame ]
    };

    new Phaser.Game(config);
})();
