precision mediump float;

#extension GL_OES_standard_derivatives : enable

uniform float time;
uniform vec2 mouse;
uniform vec2 resolution;

uniform sampler2D keyboard;
uniform sampler2D backbuffer;

const float PI = 3.14159265359;

bool isKeyPressed(float key) {
    return texture2D(keyboard, vec2(key/256.0, 0.0)).x != 0.0;
}

mat3 cameraMatrix() {
    vec2 mo = -150.0 + 300.0*(mouse.xy);
    mo.y = -mo.y;
    vec2 rad = mo*PI/180.;
    
    vec2 s = sin(rad);
    vec2 c = cos(rad);
    
    vec3 f = vec3(c.y*s.x, -s.y, c.y*c.x);
    vec3 r = vec3(c.x, 0, -s.x);
    vec3 u = normalize(cross(f, r));
    
    return mat3(r, u, f);
}

float de(vec3 p) {
    return min(length(p) - 1.0, p.y + 1.0);
}

void main( void ) {
    vec2 t = (gl_FragCoord.xy/resolution);
    if( t.y < 0.01) { 
        vec3 pos = texture2D(backbuffer, vec2(0)).xyz;
        mat3 cam = cameraMatrix();
        
        if(isKeyPressed(87.0)) pos += cam[2]*0.1;
        if(isKeyPressed(83.0)) pos -= cam[2]*0.1;
        if(isKeyPressed(65.0)) pos -= cam[0]*0.1;       
        if(isKeyPressed(68.0)) pos += cam[0]*0.1;
        if(isKeyPressed(32.0)) pos += cam[1]*0.1;
        if(isKeyPressed(16.0)) pos -= cam[1]*0.1;
        
        gl_FragColor = vec4(pos, 1);
    } else {
        vec2 p = -1.0 + 2.0*gl_FragCoord.xy/resolution;
        p.x *= resolution.x/resolution.y;
        
        
        vec3 ro = texture2D(backbuffer, vec2(0)).xyz + vec3(0, 0, -3);
        vec3 rd = normalize(cameraMatrix()*vec3(p, 1.97));
        
        float t = 0.0;
        for(int i = 0; i < 100; i++) {
            float d = de(ro + rd*t);
            if(d < 0.001 || t >= 10.0) break;
            t += d;
        }
        
        vec3 col = vec3(0);
        if(t < 10.0) {
            vec3 pos = ro + rd*t;
            vec2 eps = vec2(0.001, 0.0);
            vec3 nor = normalize(vec3(
                de(pos + eps.xyy) - de(pos - eps.xyy),
                de(pos + eps.yxy) - de(pos - eps.yxy),
                de(pos + eps.yyx) - de(pos - eps.yyx)
            ));
            
            col = 0.2*vec3(1);
            col += 0.7*  clamp(dot(normalize(vec3(0.8, 0.7, -0.6)), nor), 0.0, 1.0);
        }
        
        gl_FragColor = vec4(col, 1);
    }
}