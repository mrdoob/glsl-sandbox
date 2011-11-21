
require 'rubygems'
require 'sinatra'
require 'mongo'
require 'json'
require 'erb'

require 'pp'

configure do
    set :public_folder, 'public'

    GALLERY=ERB.new(File.read('server/assets/gallery.html'))

    uri = URI.parse(ENV['MONGOHQ_URL'])
    conn = Mongo::Connection.from_uri(ENV['MONGOHQ_URL'])
    db = conn.db(uri.path.gsub(/^\//, ''))
    VERSIONS=db.collection('versions')
    CODE=db.collection('code')
    COUNTERS=db.collection('counters')

    # initialize counters
    code=COUNTERS.find_one({:_id => 'code'})
    if !code
        COUNTERS.insert({
            :_id => 'code',
            :counter => 0
        })
    end

    EFFECTS_PER_PAGE=50
end

class Effects
    def initialize(effects, extra={})
        @effects=effects
        @extra={
            :page => 0,
            :count => 0,
            :effects_per_page => EFFECTS_PER_PAGE
        }.merge(extra)
    end

    def bind
        binding
    end

    def effects
        @effects
    end

    def extra
        @extra
    end

    def previous_page
        if @extra[:page]>0
            @extra[:page]-1
        end
    end

    def next_page
        if @extra[:count]>=((@extra[:page]+1)*@extra[:effects_per_page])
            @extra[:page]+1
        end
    end
end


def increment_code_counter
    counter=COUNTERS.find_and_modify({
        :query => {:_id => 'code'},
        :update => {'$inc' => {:counter => 1}}
    })

    counter['counter']
end

def save_version(code_id, code)
    time=Time.now
    code_data=JSON.parse(code)

    data={
        :created_at => time,
        :code => code_data['code']
    }

    CODE.find_and_modify({
        :query => { :_id => code_id },
        :update => {
            '$set' => {
                :modified_at => time,
                :image => code_data['image']
            },
            '$push' => { :versions => data }
        }
    })
end

get '/' do
    if(params[:page])
        page=params[:page].to_i
    else
        page=0
    end

    count=CODE.count();

    effects=CODE.find({}, {
        :sort => [:modified_at, 'descending'],
        :limit => EFFECTS_PER_PAGE,
        :skip => page*EFFECTS_PER_PAGE
    })

    ef=Effects.new(effects,
        :page   => page,
        :count  => count
    )

    GALLERY.result(ef.bind)
end

# assets

get '/new' do
    send_file 'static/index.html'
end

get %r{^.*/js/jquery.js$} do
    send_file 'server/assets/jquery.js'
end

get %r{^.*/js/helpers.js$} do
    send_file 'server/assets/helpers.js'
end

get %r{^.*/js/lzma.js$} do
    "\n"
end

get %r{^/(\d+)(/(\d+))?$} do
    send_file 'static/index.html'
end

get %r{/item/(\d+)(/(\d+))?} do
    code_id=params[:captures][0].to_i
    if params[:captures][1]
        version_id=params[:captures][2].to_i
    else
        version_id=nil
    end

    code=CODE.find_one({:_id => code_id})

    if version_id
        item=code['versions'][version_id]
    else
        item=code['versions'].last
    end

    if code['user']
        user=code['user']
    else
        user=false
    end

    if item
        {
            :code => item['code'],
            :user => user
        }.to_json
    else
        {
            :code => '// item not found',
            :user => false
        }.to_json
    end
end

post %r{^/(new)$} do
    counter=increment_code_counter
    body=request.body.read

    code_data=JSON.parse(body)

    data={
        :_id => counter,
        :created_at => Time.now,
        :modified_at => Time.now,
        :versions => [],
        :user => code_data['user']
    }

    CODE.insert(data)

    save_version(counter, body)

    "#{counter}/0"
end


post  %r{^/(\d+)(/(\d+))?$} do
    code_id=params[:captures][0].to_i
    body=request.body.read

    save_version(code_id, body)

    code=CODE.find_one({ :_id => code_id })

    version=code['versions'].length-1

    "#{code_id}/#{version}"
end



