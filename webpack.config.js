// initial structure from: https://github.com/patternfly/patternfly-demo-app/blob/master/build/webpack.config.js
// more from: https://github.com/webpack-contrib/mini-css-extract-plugin
const webpack = require('webpack');
const CopyWebpackPlugin = require('copy-webpack-plugin');
const MiniCssExtractPlugin = require("mini-css-extract-plugin");
const WriteFilesPlugin = require('write-file-webpack-plugin');
const OptimizeCssAssetsPlugin = require('optimize-css-assets-webpack-plugin');
const TerserJSPlugin = require('terser-webpack-plugin');
const BabelEnvDeps = require('webpack-babel-env-deps');
const CompressionPlugin = require('compression-webpack-plugin');
const BundleAnalyzerPlugin = require('webpack-bundle-analyzer').BundleAnalyzerPlugin;
const LodashModuleReplacementPlugin = require('lodash-webpack-plugin');

// path to assets
const ASSET_PATH = process.env.ASSET_PATH || '/';

module.exports = (env, argv) => {
    // get dev mode
    const devMode = argv.mode === "development";
    const statsMode = argv.stats === "true";

    // output
    console.log("build flags dev=" + devMode + ", stats=" + statsMode);

    // return dynamic config based on options
    return {
        name: "gudgeon",

        context: __dirname + "/web/",

        entry: {
            "gudgeon": "./app/js/gudgeon-index.js"
        },

        cache: devMode,

        output: {
            publicPath: ASSET_PATH,
            path: __dirname + "/web/static",
            filename: "js/[name].bundle.js",
            hotUpdateChunkFilename: 'hot/[id].[hash].hot-update.js',
            hotUpdateMainFilename: 'hot/[hash].hot-update.json'
        },

        // having consistency issues with watching for file changes so this
        // came from stackoverflow initially (https://stackoverflow.com/questions/26708205/webpack-watch-isnt-compiling-changed-files)
        watchOptions: {
            poll: true,
            ignored: ['node_modules']
        },

        optimization: {
            // optimization flags
            usedExports: true,
            minimize: true,
            concatenateModules: true,
            mergeDuplicateChunks: true,
            providedExports: true,

            // various minimizers
            minimizer: [
                new TerserJSPlugin(),
                new OptimizeCssAssetsPlugin({
                    cssProcessor: require('cssnano'),
                    cssProcessorPluginOptions: {
                        preset: ['default', {discardComments: {removeAll: true}}],
                    },
                    canPrint: true
                })
            ]
        },

        plugins: [
            // Avoid publishing files when compilation failed:
            new webpack.NoEmitOnErrorsPlugin(),

            //HMR
            new webpack.HotModuleReplacementPlugin(),

            //make webpack ignore moment locale require: https://github.com/moment/moment/issues/2517
            new webpack.IgnorePlugin(/^\.\/locale$/, /moment$/),

            //creates distribution css file rather than inlining styles
            new MiniCssExtractPlugin({
                // Options similar to the same options in webpackOptions.output
                // both options are optional
                filename: "css/" + (devMode ? '[name].bundle.css' : '[name].[hash].bundle.css'),
                chunkFilename: "css/" + (devMode ? '[id].bundle.css' : '[id].[hash].bundle.css'),
            }),

            new CopyWebpackPlugin([
                // copy favicon.ico, basically
                {
                    from: {glob: __dirname + '/web/app/ico/*.ico'},
                    to: __dirname + '/web/static/',
                    flatten: true
                },

                // copy raw html if needed
                {
                    from: {glob: __dirname + '/web/app/html/*.html'},
                    to: __dirname + '/web/static/',
                    flatten: true
                },

                // copy html templates
                {
                    from: {glob: __dirname + '/web/app/html/*.tmpl'},
                    to: __dirname + '/web/static/',
                    flatten: true
                },

                // copy images from app
                {
                    from: {glob: __dirname + '/web/app/img/*'},
                    to: './img',
                    flatten: true
                },

                // copy patternfly assets for app
                {
                    from: {glob: './node_modules/@patternfly/patternfly/assets/images/*.*'},
                    to: './img',
                    flatten: true
                },
            ]),

            // writes files on changes to src
            new WriteFilesPlugin(),

            // bundle analyzer
            new BundleAnalyzerPlugin({
                analyzerMode: devMode ? 'server' : (statsMode ? 'static' : 'disabled'),
                reportFilename: "../../build/report.html",
                generateStatsFile: false,
                openAnalyzer: devMode,
            }),

            // add compression plugin
            new CompressionPlugin({
                test: /\.(js|css|html|svg|eot|ttf|woff|woff2)$/,
                deleteOriginalAssets: !devMode && !statsMode,
                threshold: 4092,
                minRatio: 0.90
            }),

            // lodash module replacement
            new LodashModuleReplacementPlugin({
                "currying": true,
                "flattening": true,
                "paths": true,
                "placeholders": true,
                "shorthands": true
            })
        ],

        module: {
            rules: [
                //js loader
                {
                    // Only run `.js` and `.jsx` files through Babel
                    test: /\.jsx?$/,
                    // don't load modules that don't need transpilation
                    exclude: [
                        BabelEnvDeps.exclude()
                    ],
                    use: [
                        {
                            loader: "babel-loader",
                            options: {
                                plugins: [
                                    "lodash",
                                    '@babel/plugin-proposal-class-properties',
                                    '@babel/plugin-syntax-dynamic-import',
                                    '@babel/plugin-transform-typescript',
                                    'babel-plugin-typescript-to-proptypes',
                                    '@babel/plugin-proposal-export-default-from',
                                    '@babel/plugin-proposal-object-rest-spread',
                                ],
                                presets: [
                                    '@babel/preset-react',
                                    '@babel/preset-env'
                                ]
                            }
                        }
                    ]
                },

                // load css
                {
                    test: /\.css$/,
                    use: [
                        {
                            loader: MiniCssExtractPlugin.loader,
                            options: {
                                hmr: devMode
                            }
                        },
                        {
                            loader: "css-loader",
                            options: {
                                sourceMap: devMode,
                                importLoaders: 1
                            }
                        },
                        {
                            loader: "postcss-loader",
                            options: {
                                sourceMap: devMode
                            }
                        }
                    ]
                },

                // font/image url loaders
                {
                    test: /\.svg(\?v=\d+\.\d+\.\d+)?$/,
                    use: [
                        {
                            loader: 'svg-url-loader',
                            options: {
                                limit: 16 * 1024,
                                mimetype: "image/svg+xml",
                                name: "img/[name].[ext]"
                            }
                        }
                    ]
                },
                {
                    test: /\.(woff)(\?v=\d+\.\d+\.\d+)?$/,
                    use: [
                        {
                            loader: 'url-loader',
                            options: {
                                limit: 16 * 1024,
                                mimetype: "application/font-woff",
                                name: "fonts/[name].[ext]"
                            }
                        }
                    ]
                },
                {
                    test: /\.(woff2)(\?v=\d+\.\d+\.\d+)?$/,
                    use: [
                        {
                            loader: 'url-loader',
                            options: {
                                limit: 16 * 1024,
                                mimetype: "application/font-woff2",
                                name: "fonts/[name].[ext]"
                            }
                        }
                    ]
                },
                {
                    test: /\.ttf(\?v=\d+\.\d+\.\d+)?$/,
                    use: [
                        {
                            loader: 'url-loader',
                            options: {
                                limit: 16 * 1024,
                                mimetype: "application/octet-stream",
                                name: "fonts/[name].[ext]"
                            }
                        }
                    ]
                },
                {
                    test: /\.eot(\?v=\d+\.\d+\.\d+)?$/,
                    use: [
                        {
                            loader: 'url-loader',
                            options: {
                                limit: 16 * 1024,
                                mimetype: "application/vnd.ms-fontobject",
                                name: "fonts/[name].[ext]"
                            }
                        }
                    ]
                },
                {
                    test: /\.(png|jpe?g|gif)(\?\S*)?$/,
                    use: [
                        {
                            loader: 'url-loader',
                            options: {
                                limit: 8 * 1024,
                                name: "img/[name].[ext]"
                            }
                        }
                    ]
                }
            ]
        }
    };
};