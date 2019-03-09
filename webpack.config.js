// initial structure from: https://github.com/patternfly/patternfly-demo-app/blob/master/build/webpack.config.js
// more from: https://github.com/webpack-contrib/mini-css-extract-plugin
var webpack = require('webpack');
var CopyWebpackPlugin = require('copy-webpack-plugin');
var MiniCssExtractPlugin = require("mini-css-extract-plugin");
var WriteFilesPlugin = require('write-file-webpack-plugin');
var OptimizeCssAssetsPlugin = require('optimize-css-assets-webpack-plugin');

const devMode = process.env.NODE_ENV !== 'production';

module.exports = {
    
    context: __dirname + "/web/",

    entry: {
        "gudgeon": "./app/js/gudgeon-index.js"
    },

    output: {
        path: __dirname + "/web/static",
        filename: "js/[name].bundle.js",
        hotUpdateChunkFilename: 'hot/[id].[hash].hot-update.js',
        hotUpdateMainFilename: 'hot/[hash].hot-update.json'        
    },

    externals: {
        // require("jquery") is external and available on the global var jQuery
        "jquery": "jQuery",
        "jquery": "$"
    },

    plugins: [
        // Avoid publishing files when compilation failed:
        new webpack.NoEmitOnErrorsPlugin(),

        //HMR
        new webpack.HotModuleReplacementPlugin(),

        //make webpack ignore moment locale require: https://github.com/moment/moment/issues/2517
        new webpack.IgnorePlugin(/^\.\/locale$/, /moment$/),

         //global jquery is provided to any webpack modules 
        new webpack.ProvidePlugin({
            $: "jquery",
            jQuery: "jquery",
            "window.jQuery": "jquery"
        }),

        //creates distribution css file rather than inlining styles
        new MiniCssExtractPlugin({
            // Options similar to the same options in webpackOptions.output
            // both options are optional
            filename: "css/" + (devMode ? '[name].bundle.css' : '[name].[hash].bundle.css'),
            chunkFilename: "css/" + ( devMode ? '[id].bundle.css' : '[id].[hash].bundle.css'),
        }),

        new OptimizeCssAssetsPlugin({
            assetNameRegExp: /\.bundle\.css$/g,
            cssProcessor: require('cssnano'),
            cssProcessorPluginOptions: {
                preset: ['default', { discardComments: { removeAll: true } }],
            },
            canPrint: true
        }),

        //copy patternfly assets for demo app
        new CopyWebpackPlugin([
            {
                from: { glob: __dirname + '/web/app/html/*.html'},
                to: __dirname + '/web/static/',
                flatten: true
            },
            {
                from: { glob: __dirname + '/web/app/html/*.tmpl'},
                to: __dirname + '/web/static/',
                flatten: true
            },            
            {
                from: { glob: './node_modules/@patternfly/patternfly/assets/images/*.*'},
                to: './img',
                flatten: true
            },            
        ]),

        //writes files on changes to src
        new WriteFilesPlugin()        
    ],

    module: {
        rules: [
            //js loader
            {
                // Only run `.js` and `.jsx` files through Babel
                test: /\.jsx?$/,

                exclude: /node_modules/,

                use: [
                    {
                        loader: "babel-loader",
                        options: {
                            plugins: ['transform-runtime', '@patternfly/react-styles/babel'],
                            presets: ['react', 'es2015', 'stage-0']
                        }
                    }
                ]
            },

            // bundle LESS and CSS auto-generating -vendor-prefixes
            {
                test: /\.css$/,
                use: [
                    { 
                        loader: "style-loader" 
                    },
                    {
                        loader: MiniCssExtractPlugin.loader,
                    },
                    {                  
                        loader: "css-loader",
                        options: {
                            sourceMap: true
                        }
                    }
                ]
            },
            {
                test: /\.less$/,
                use: [
                    { 
                        loader: "style-loader" 
                    },
                    {
                        loader: MiniCssExtractPlugin.loader,
                    },
                    {
                        loader: "css-loader",
                        options: {
                            sourceMap: true
                        }
                    },
                    {
                        loader: "autoprefixer-loader"
                    },
                                                            {
                        loader: "less-loader"
                    }
                ]
            },

            //font/image url loaders
            {
                test: /\.svg(\?v=\d+\.\d+\.\d+)?$/,
                use: [
                    {
                        loader: 'url-loader',
                        options: {
                            limit: 65000,
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
                            limit: 65000,
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
                            limit: 65000,
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
                            limit: 65000,
                            mimetype: "application/octet-stream",
                            name: "fonts/[name].[ext]"
                        }
                    }
                ]                
            },
            {
                test:  /\.eot(\?v=\d+\.\d+\.\d+)?$/,
                use: [
                    {
                        loader: 'url-loader',
                        options: {
                            limit: 65000,
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
                            limit: 100000,
                            name: "img/[name].[ext]"
                        }
                    }   
                ]                
            }            
        ]
    }
};