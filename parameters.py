import json
import urllib
import sys
import time

#We import the json library because we will be taking json data and transforming it to
#python objects.  We import the urllib as we will be grabbing the url of the json data
#and will be translating them through the json library.  Sys library is to utilize the argv
#and exit command, and we use time to establish the date in the program for the TIL facts.

def getjsonData(limit_amount):
    url = "http://www.reddit.com/r/todayilearned/new/.json?limit=" + str(limit_amount)
    response = urllib.urlopen(url)
    data = json.loads(response.read())
    return data
    
#We create a function called getjsonData with the argument of limit_amount so we know how
#many TIL facts we're going to be printing and the limitation of it.  This functionis what
#grabs the TIL facts information from the Reddit API website, and we add the limit that we're
#going to determine as a string at the end.  We create the response variable to open up this
#API and download the data, and have the json library load it into the variable with Python
#objects.  We have it return the data information.

def facts(data):
     for temp in data['data']['children']:
        title = temp['data']['title']
        #replace TIL with a blank string
        title = title.replace("TIL","")
        print title + "\n"

#This function prints the facts.  We take the data we used from the previous function and 
#place it into the argument.  We create a for statement and create a variable called temp,
#which grabs the "children" of the data and stores it.  We create title variable and get the
#title from the data block.  To separate each fact in a clean and readable manner, we use the
#replace module so we can replace the "TIL" that would be replaced with just a blank string.
#We also give a new line to each printed title.
            
limit = 25

#We assign the global limit to 25 arguments, if we had more variables.

argument_list = sys.argv

#The difference of this github is that this is using the argv module in sys.  Here, we create
#the argument_list to hold the argument variables.  This means that the argv list will be called
#on by calling the variable argument_list.  Since argument 0 placement is the script, and argument
#1 in argv would be the first variable, we're able to set the limit of the if statement to those
#two arguments.

if len(argument_list) > 1:
    limit = argument_list[1]

json_data = getjsonData(limit)

#We create the json_data variable, and in it we call the getjsonData function, and institute the limit
#in the argument, which in the above argv, we have stated we just want the script parameters.py to run
#and to print the title.


print "Printing Reddit's TIL threads of " + time.strftime("%d/%m/%Y")
facts(json_data) 
print "\n"

while(True):
    user_input = raw_input("Would you like to close? ")

    if user_input == 'Yes' or user_input == 'yes' or user_input == 'Y' or user_input == 'y':
       sys.exit()
       
    elif user_input == 'No' or user_input == 'no' or user_input == 'N' or user_input == 'n':
        facts(getjsonData(limit))
    
    else:
        print "Error, please enter yes or no."

#We greet the user, let them know that they are printing the TIL facts from Reddit from whatever
#date, and we give them the option to close or not.  If they chose not to, we generate newer TIL
#facts that could be published between when they finished their first 25 facts they've read.