
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

    def image(effect)
        "http://res.cloudinary.com/#{CLOUDINARY_PROJECT}/image/upload/"<<
            "#{IMAGE_PREFIX}#{effect['_id']}.png"
    end
end

class GlslDatabase
    def initialize
        connect_database
        initialize_counter
    end

    def increment_code_counter
        counter=@counters.find_and_modify({
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

        @code.find_and_modify({
            :query => { :_id => code_id },
            :update => {
                '$set' => {
                    :modified_at => time
                },
                '$push' => { :versions => data }
            }
        })

        Cloudinary::Uploader.upload(
            code_data['image'],
            :public_id => IMAGE_PREFIX+code_id.to_s)

        code=$glsl.get_code(code_id)

        version=code['versions'].length-1

        "#{code_id}.#{version}"
    end

    def save_effect(code)
        code_data=JSON.parse(code)

        if code_data['code_id'] && !code_data['code_id'].empty?
            m=code_data['code_id'].match(/^(\d+)(\.\d+)?/)
            if m
                save_version(m[1].to_i, code)
            else
                save_new_effect(code)
            end
        else
            save_new_effect(code)
        end
    end

    def save_new_effect(code)
        counter=increment_code_counter

        code_data=JSON.parse(code)

        data={
            :_id => counter,
            :created_at => Time.now,
            :modified_at => Time.now,
            :versions => [],
            :user => code_data['user']
        }

        if code_data['parent']
            m=code_data['parent'].match(%r{^(\d+)(\.(\d+))?})
            data[:parent] = m[1].to_i if m
            data[:parent_version] = m[3].to_i if m && m[3]
        end

        @code.insert(data)

        save_version(counter, code)
    end

    def get_page(page=0, effects_per_page=50)
        count=@code.count()

        effects=@code.find({}, {
            :sort => [:modified_at, 'descending'],
            :limit => effects_per_page,
            :skip => page*effects_per_page
        })

        ef=Effects.new(effects,
            :page   => page,
            :count  => count
        )

        ef
    end

    def get_code(id)
        @code.find_one({:_id => id})
    end

    def get_code_json(id, version=nil)
        code=get_code(id)

        if code
            if version
                item=code['versions'][version]
            else
                item=code['versions'].last
            end
            
            if code['user']
                user=code['user']
            else
                user=false
            end

            parent=nil
            if code['parent']
                parent="/e##{code['parent']}"
                parent+=".#{code['parent_version']}" if code['parent_version']
            end
        else
            item=nil
        end

        if item
            data={
                :code => item['code'],
                :user => user,
                :parent => parent
            }.to_json
        else
            {
                :code => '// item not found',
                :user => false
            }.to_json
        end
    end

private

    def connect_database
        uri = URI.parse(ENV['MONGOHQ_URL'])
        conn = Mongo::Connection.from_uri(ENV['MONGOHQ_URL'])
        @db = conn.db(uri.path.gsub(/^\//, ''))

        @versions=@db.collection('versions')
        @code=@db.collection('code')
        @counters=@db.collection('counters')
    end

    def initialize_counter
        code=@counters.find_one({:_id => 'code'})
        if !code
            @counters.insert({
                :_id => 'code',
                :counter => 0
            })
        end
    end
end
