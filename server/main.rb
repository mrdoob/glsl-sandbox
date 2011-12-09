
require 'rubygems'
require 'sinatra'
require 'mongo'
require 'json'
require 'erb'

$: << './server'

require 'model'

require 'pp'

configure do
    set :public_folder, 'server/assets'

    GALLERY=ERB.new(File.read('server/assets/gallery.html'))

    $glsl=GlslDatabase.new

    EFFECTS_PER_PAGE=50
end

get '/' do
    if(params[:page])
        page=params[:page].to_i
    else
        page=0
    end

    ef=$glsl.get_page(page, EFFECTS_PER_PAGE)

    GALLERY.result(ef.bind)
end

# assets

get '/new' do
    send_file 'static/index.html'
end

get '/e' do
    send_file 'static/index.html'
end

get %r{^.*/js/jquery.js$} do
    send_file 'server/assets/jquery.js'
end

get %r{^.*/js/helpers.js$} do
    send_file 'server/assets/helpers.js'
end

get %r{^.*/js/glsl.js$} do
    send_file 'static/js/glsl.js'
end

get %r{^.*/js/codemirror.js$} do
    send_file 'static/js/codemirror.js'
end

get %r{^.*/css/codemirror.css$} do
    send_file 'static/css/codemirror.css'
end

get %r{^.*/css/default.css$} do
    send_file 'static/css/default.css'
end

get %r{^.*/js/lzma.js$} do
    "\n"
end

get %r{^/(\d+)(/(\d+))?$} do
    send_file 'static/index.html'
end

get %r{/item/(\d+)([/.](\d+))?} do
    code_id=params[:captures][0].to_i
    if params[:captures][1]
        version_id=params[:captures][2].to_i
    else
        version_id=nil
    end

    $glsl.get_code_json(code_id, version_id)
end

post %r{^/(new)$} do
    body=request.body.read

    $glsl.save_new_effect(body)
end


post  %r{^/(\d+)(/(\d+))?$} do
    code_id=params[:captures][0].to_i
    body=request.body.read

    $glsl.save_version(code_id, body)

    code=$glsl.get_code(code_id)

    version=code['versions'].length-1

    "#{code_id}/#{version}"
end

post '/e' do
    body=request.body.read
    $glsl.save_effect(body)
end




