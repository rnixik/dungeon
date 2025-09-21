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
            mode: Phaser.Scale.RESIZE,
            autoCenter: Phaser.Scale.CENTER_BOTH,
            min: {
                width: 320,
                height: 240
            },
            max: {
                width: 800,
                height: 600
            }
        },
        scene: [ sceneConfigBoot, sceneConfigPreloader, sceneConfigMainMenu, sceneConfigGame ]
    };

    new Phaser.Game(config);
})();
