import sys
import re
import os

def replace_kw_calls(match):
    s = match.group(1).strip('"')
    prefix, suffix = "", ""
    
    if s.endswith("?"):
        prefix = "Is"
        s = s[:-1]
    elif s.endswith("!"):
        suffix = "Bang"
        s = s[:-1]

    s = re.sub(r'[-/]', ' ', s).title().replace(' ', '')
    return f"KW{prefix}{s}{suffix}"

def main():
    if len(sys.argv) < 2:
        print("Usage: python replace_kw.py <input_file.go>")
        sys.exit(1)

    input_file = sys.argv[1]
    output_file = os.path.splitext(input_file)[0] + "_replaced.go"

    kw_pattern = re.compile(r'kw\(\s*"([^"]*)"\s*\)')

    try:
        with open(input_file, 'r') as infile, open(output_file, 'w') as outfile:
            for line in infile:
                replaced_line = kw_pattern.sub(replace_kw_calls, line)
                outfile.write(replaced_line)
    except FileNotFoundError:
        print(f"File '{input_file}' not found.")
    except Exception as e:
        print(f"Error occurred: {e}")
    else:
        print(f"Replacements made successfully. Output saved to '{output_file}'")

if __name__ == "__main__":
    main()
