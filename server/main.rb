
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

get '/e' do
    send_file 'static/index.html'
end

=begin
get '/new' do
    send_file 'static/index.html'
end


get %r{^/(\d+)(/(\d+))?$} do
    send_file 'static/index.html'
end
=end

get %r{/item/(\d+)([/.](\d+))?} do
    code_id=params[:captures][0].to_i
    if params[:captures][1]
        version_id=params[:captures][2].to_i
    else
        version_id=nil
    end

    $glsl.get_code_json(code_id, version_id)
end

post '/e' do
    body=request.body.read
    $glsl.save_effect(body)
end




