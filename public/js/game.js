(function() {
    const config = {
        type: Phaser.AUTO,
        width: 800,
        height: 600,
        pixelArt: true,
        physics: {
            default: 'arcade',
            arcade: {
                debug: false
            }
        },
        scale: {
            mode: Phaser.Scale.FIT,
            autoCenter: Phaser.Scale.CENTER_BOTH,
            min: {
                width: 320,
                height: 240
            },
            max: {
                width: 1200,
                height: 900
            }
        },
        scene: [ sceneConfigBoot, sceneConfigPreloader, sceneConfigGame ]
    };

    new Phaser.Game(config);
})();
